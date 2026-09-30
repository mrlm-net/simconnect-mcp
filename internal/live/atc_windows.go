//go:build windows

package live

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mrlm-net/simconnect/pkg/airport"
	"github.com/mrlm-net/simconnect/pkg/calc"
	"github.com/mrlm-net/simconnect/pkg/traffic"
)

// Airborne ATC (simconnect v0.16) in the live runtime, run every tick:
//
//   - a tower per runway (traffic.RunwayController) clears our departures'
//     line-up and take-off and our aircraft's runway crossings — those not
//     spawned holding for clearances — and sends an arrival around when the
//     runway is not free on short final;
//   - a landing sequence per runway end (traffic.ApproachSequencer) of our
//     arrivals and the other traffic on final; our arrivals lose their
//     delays by speed, then a longer downwind, and hold at the STAR fix
//     (stacked) what is still left;
//   - every instruction is kept as an ATC message for get_atc_log.

// Sequencing figures, as on the library's airport map.
const (
	atcAbsorbFrom    = 30 * time.Second // a delay is absorbed from this on…
	atcAbsorbEvery   = 90 * time.Second // …at most this often per arrival
	atcHoldFrom      = time.Minute      // left after absorbing: hold
	atcHoldRelease   = time.Minute      // a holding arrival leaves at this delay
	atcHoldFixNM     = 15.0             // the hold fix at least this far out
	atcHoldBaseFt    = 6000.0           // the stack's lowest level
	atcFinalNM       = 20.0             // arrivals this close are on the tower's list
	atcSlowBy        = time.Minute      // approach_instruction "slow"
	atcLogSize       = 200
	atcRunwayMarginM = 15.0
)

// ATCMessage is an instruction given by the runtime's controllers.
type ATCMessage struct {
	At       time.Time `json:"at"`
	ICAO     string    `json:"icao"`
	Callsign string    `json:"callsign"`
	Text     string    `json:"text"`
}

// RunwaySequence is the landing sequence of a runway end.
type RunwaySequence struct {
	ICAO       string                     `json:"icao"`
	Runway     string                     `json:"runway"`
	Conditions traffic.ApproachConditions `json:"conditions"`
	Sequence   []traffic.SequenceEntry    `json:"sequence"`
}

// RunwayUser is who uses a runway now, as its tower sees it.
type RunwayUser struct {
	Runway   string `json:"runway"`
	Callsign string `json:"callsign"`
	Phase    string `json:"phase"`
	Ours     bool   `json:"ours"`
	Waiting  string `json:"waiting,omitempty"`
}

// ErrNotSequenced: approach_instruction for an aircraft not in a landing
// sequence (not one of our arrivals in the air).
var ErrNotSequenced = errors.New("not in a landing sequence")

type atcState struct {
	towers   map[string]*traffic.RunwayController  // by "ICAO 06/24"
	seqs     map[string]*traffic.ApproachSequencer // by "ICAO 06"
	stacks   map[string]*traffic.HoldStack         // by "ICAO lat lon"
	given    map[string]bool                       // clearances given: "cs action"
	absorbed map[string]time.Time
	users    map[string][]RunwayUser // by ICAO, the last tick's
	log      []ATCMessage
}

func newATCState() *atcState {
	return &atcState{towers: map[string]*traffic.RunwayController{}, seqs: map[string]*traffic.ApproachSequencer{},
		stacks: map[string]*traffic.HoldStack{}, given: map[string]bool{}, absorbed: map[string]time.Time{}, users: map[string][]RunwayUser{}}
}

func (a *atcState) say(now time.Time, icao, cs, text string) {
	a.log = append(a.log, ATCMessage{At: now, ICAO: icao, Callsign: cs, Text: text})
	if len(a.log) > atcLogSize {
		a.log = a.log[len(a.log)-atcLogSize:]
	}
}

var runwayPhaseNames = map[traffic.RunwayPhase]string{traffic.RunwayHoldingShort: "holding short", traffic.RunwayLinedUp: "lined up",
	traffic.RunwayRolling: "on the runway", traffic.RunwayAirborne: "airborne", traffic.RunwayFinal: "final"}

// runwayNamed is the runway of an end or runway name ("06", "06/24").
func runwayNamed(l *airport.Layout, name string) (airport.Runway, bool) {
	for _, r := range l.Runways {
		if r.Name() == name || r.Primary.Name == name || r.Secondary.Name == name {
			return r, true
		}
	}
	return airport.Runway{}, false
}

// onRunway reports p on the runway's surface (and a margin).
func onRunway(r airport.Runway, p airport.LatLon) bool {
	a, b := r.Primary.Threshold, r.Secondary.Threshold
	along := calc.AlongTrackMeters(a.Lat, a.Lon, b.Lat, b.Lon, p.Lat, p.Lon)
	cross := math.Abs(calc.CrossTrackMeters(a.Lat, a.Lon, b.Lat, b.Lon, p.Lat, p.Lon))
	return along > -100 && along < r.Length+100 && cross < r.Width/2+atcRunwayMarginM
}

// finalOf is the runway end an aircraft at p heading hdg is on the final
// of (within atcFinalNM, 1.5 NM of the centreline, heading within 30°) and
// its distance to the threshold.
func finalOf(l *airport.Layout, p airport.LatLon, hdg float64) (airport.RunwayEnd, float64, bool) {
	for _, r := range l.Runways {
		for _, e := range []airport.RunwayEnd{r.Primary, r.Secondary} {
			t := e.Threshold
			d := calc.HaversineNM(p.Lat, p.Lon, t.Lat, t.Lon)
			if d > atcFinalNM || math.Abs(math.Mod(hdg-e.Heading+540, 360)-180) > 30 {
				continue
			}
			farLat, farLon := calc.DisplaceByHeading(t.Lat, t.Lon, e.Heading+180, 20*1852)
			if math.Abs(calc.CrossTrackMeters(t.Lat, t.Lon, farLat, farLon, p.Lat, p.Lon)) < 1.5*1852 &&
				calc.AlongTrackMeters(t.Lat, t.Lon, farLat, farLon, p.Lat, p.Lon) > 0 {
				return e, d, true
			}
		}
	}
	return airport.RunwayEnd{}, 0, false
}

// tickATCLocked runs the towers and the landing sequences (r.mu held).
func (r *Runtime) tickATCLocked(now time.Time) {
	t := r.traffic
	if t == nil {
		return
	}
	if t.atc == nil {
		t.atc = newATCState()
	}
	a := t.atc
	t.tmu.Lock()
	flights := make([]*flight, 0, len(t.flights))
	for _, f := range t.flights {
		flights = append(flights, f)
	}
	views := map[*flight]FlightView{}
	for _, f := range flights {
		views[f] = f.view
	}
	t.tmu.Unlock()
	graphs := map[string]*airport.Graph{}
	graph := func(icao string) *airport.Graph {
		if g, ok := graphs[icao]; ok {
			return g
		}
		g, err := r.cache.Graph(icao)
		if err != nil {
			g = nil
		}
		graphs[icao] = g
		return g
	}
	others := t.picture.Aircraft()
	a.runTowers(r, now, flights, views, graph, others)
	a.runSequences(r, now, flights, views, graph, others)
}

// runTowers gives the runway clearances (r.mu held).
func (a *atcState) runTowers(r *Runtime, now time.Time, flights []*flight, views map[*flight]FlightView, graph func(string) *airport.Graph, others []traffic.TrackedAircraft) {
	type key struct{ icao, rwy string }
	users := map[key][]traffic.RunwayUser{}
	ours := map[string]*flight{}
	icaos := map[string]bool{}
	for _, f := range flights {
		v := views[f]
		g := graph(v.ICAO)
		if g == nil || v.Done {
			continue
		}
		icaos[v.ICAO] = true
		l := g.Layout
		own, ok := runwayNamed(l, v.Runway)
		if !ok {
			continue
		}
		if v.State != "holding short" {
			a.forgetCrossings(v.Callsign)
		}
		u := traffic.RunwayUser{Callsign: v.Callsign, Wake: traffic.WakeFor(v.Model), Route: v.Procedure, Other: f.held}
		rk := key{v.ICAO, own.Name()}
		switch {
		case v.State == "taxiing" && v.OnGround:
			for _, rw := range l.Runways {
				if onRunway(rw, v.Position) {
					u.Phase, u.Crossing = traffic.RunwayRolling, true
					users[key{v.ICAO, rw.Name()}] = append(users[key{v.ICAO, rw.Name()}], u)
				}
			}
			continue
		case v.State == "holding short" && v.HoldingShortOf != "" && v.HoldingShortOf != own.Name():
			x, ok := runwayNamed(l, v.HoldingShortOf)
			if !ok {
				continue
			}
			u.Phase, u.Crossing = traffic.RunwayHoldingShort, true
			rk = key{v.ICAO, x.Name()}
		case f.dep != nil && v.State == "holding short":
			u.Phase = traffic.RunwayHoldingShort
		case f.dep != nil && (v.State == "lining up" || v.State == "lined up"):
			u.Phase = traffic.RunwayLinedUp
		case f.dep != nil && v.State == "departing" && v.OnGround:
			u.Phase = traffic.RunwayRolling
		case f.dep != nil && v.State == "departing":
			u.Phase = traffic.RunwayAirborne
		case f.arr != nil && (v.State == "approaching" || v.State == "landing") && !v.OnGround:
			_, end, _ := l.RunwayEnd(v.Runway)
			d := calc.HaversineNM(v.Position.Lat, v.Position.Lon, end.Threshold.Lat, end.Threshold.Lon)
			if d > 3 {
				delete(a.given, v.Callsign+" goaround") // out again: another go-around may follow
			}
			if d > atcFinalNM {
				continue
			}
			u.Phase, u.Arrival, u.DistanceNM, u.GroundKts = traffic.RunwayFinal, true, d, v.GroundSpeed
		case f.arr != nil && (v.State == "landing" || v.State == "rollout" || v.State == "vacating"):
			u.Phase, u.Arrival = traffic.RunwayRolling, true
		default:
			continue
		}
		ours[v.Callsign] = f
		users[rk] = append(users[rk], u)
	}
	// Other traffic (not ours) on a runway or on a final, where ours are.
	for _, o := range others {
		if o.Ours {
			continue
		}
		for icao := range icaos {
			g := graph(icao)
			if g == nil {
				continue
			}
			name := o.Tail
			if name == "" {
				name = o.Title
			}
			u := traffic.RunwayUser{Callsign: name, Wake: traffic.WakeFor(o.Title), Other: true}
			if o.OnGround {
				for _, rw := range g.Layout.Runways {
					if onRunway(rw, o.Position) && o.GroundKts > 3 {
						u.Phase = traffic.RunwayRolling
						users[key{icao, rw.Name()}] = append(users[key{icao, rw.Name()}], u)
					}
				}
				continue
			}
			if end, d, ok := finalOf(g.Layout, o.Position, o.Heading); ok && o.AGLFt < 5000 {
				if rw, ok := runwayNamed(g.Layout, end.Name); ok {
					u.Phase, u.Arrival, u.DistanceNM, u.GroundKts = traffic.RunwayFinal, true, d, o.GroundKts
					users[key{icao, rw.Name()}] = append(users[key{icao, rw.Name()}], u)
				}
			}
		}
	}
	shown := map[string][]RunwayUser{}
	for k, list := range users {
		tk := k.icao + " " + k.rwy
		rc := a.towers[tk]
		if rc == nil {
			rc = traffic.NewRunwayController(traffic.RunwayControllerOptions{})
			a.towers[tk] = rc
		}
		c := rc.Decide(now, list)
		a.applyClearances(r, now, k.icao, k.rwy, c, ours)
		for _, u := range list {
			shown[k.icao] = append(shown[k.icao], RunwayUser{Runway: k.rwy, Callsign: u.Callsign, Phase: runwayPhaseNames[u.Phase],
				Ours: ours[u.Callsign] != nil, Waiting: c.Waiting[u.Callsign]})
		}
	}
	a.users = shown
}

// applyClearances gives the tower's clearances to ours, once each.
func (a *atcState) applyClearances(r *Runtime, now time.Time, icao, rwy string, c traffic.RunwayClearances, ours map[string]*flight) {
	give := func(cs, action, text string, do func(f *flight) error) {
		f := ours[cs]
		if f == nil || f.held || a.given[cs+" "+action] {
			return
		}
		if err := do(f); err != nil {
			a.say(now, icao, cs, fmt.Sprintf("%s, %s refused: %v", cs, action, err))
			return
		}
		a.given[cs+" "+action] = true
		if action == "takeoff" {
			a.given[cs+" lineup"] = true
		}
		a.say(now, icao, cs, cs+", "+text)
	}
	takeoff := map[string]bool{}
	for _, cs := range c.Takeoff {
		takeoff[cs] = true
	}
	end := func(cs string) string {
		if f := ours[cs]; f != nil {
			return f.view.Runway
		}
		return rwy
	}
	for _, cs := range c.LineUp {
		if takeoff[cs] {
			give(cs, "takeoff", "runway "+end(cs)+", line up, cleared for take-off", func(f *flight) error {
				f.dep.ClearToLineUp()
				return f.dep.ClearForTakeoff()
			})
			continue
		}
		give(cs, "lineup", "runway "+end(cs)+", line up and wait", func(f *flight) error { f.dep.ClearToLineUp(); return nil })
	}
	for _, cs := range c.Takeoff {
		give(cs, "takeoff", "runway "+end(cs)+", cleared for take-off", func(f *flight) error { return f.dep.ClearForTakeoff() })
	}
	for _, cs := range c.Cross {
		give(cs, "cross "+rwy, "cross runway "+rwy, func(f *flight) error {
			if f.dep != nil {
				f.dep.ClearToCross()
			} else {
				f.arr.ClearToCross()
			}
			return nil
		})
	}
	for _, cs := range c.GoAround {
		give(cs, "goaround", "go around, I say again, go around — "+c.Waiting[cs], func(f *flight) error {
			if f.arr == nil {
				return nil
			}
			if err := f.arr.GoAround(); err != nil {
				return err
			}
			a.rejoin(icao, cs)
			return nil
		})
	}
}

func (a *atcState) forgetCrossings(cs string) {
	for k := range a.given {
		if strings.HasPrefix(k, cs+" cross ") {
			delete(a.given, k)
		}
	}
}

// rejoin sequences an arrival at icao afresh (after a go-around).
func (a *atcState) rejoin(icao, cs string) {
	for k, s := range a.seqs {
		if strings.HasPrefix(k, icao+" ") {
			s.Rejoin(cs)
		}
	}
}

// runSequences feeds the landing sequences and has our arrivals lose
// their delays (r.mu held).
func (a *atcState) runSequences(r *Runtime, now time.Time, flights []*flight, views map[*flight]FlightView, graph func(string) *airport.Graph, others []traffic.TrackedAircraft) {
	type key struct{ icao, end string }
	feed := map[key][]traffic.ApproachAircraft{}
	byCS := map[string]*flight{}
	icaos := map[string]bool{}
	for _, f := range flights {
		v := views[f]
		if f.arr == nil || v.Done || v.OnGround || (v.State != "approaching" && v.State != "landing") || v.Position == (airport.LatLon{}) {
			continue
		}
		g := graph(v.ICAO)
		if g == nil {
			continue
		}
		_, end, ok := g.Layout.RunwayEnd(v.Runway)
		if !ok {
			continue
		}
		icaos[v.ICAO] = true
		byCS[v.Callsign] = f
		// On its procedure: what it still flies; on the final: straight in.
		dtg := traffic.DistanceVia(v.Position, f.arr.ProcedureRoute(), end.Threshold)
		prof := traffic.ProfileFor(v.Model)
		feed[key{v.ICAO, end.Name}] = append(feed[key{v.ICAO, end.Name}], traffic.ApproachAircraft{Callsign: v.Callsign,
			Wake: traffic.WakeFor(v.Model), DistanceToGoNM: dtg, GroundKts: v.GroundSpeed, FinalKts: prof.Approach.ApproachKts})
	}
	for _, o := range others {
		if o.Ours || o.OnGround {
			continue
		}
		for icao := range icaos {
			if g := graph(icao); g != nil {
				if end, d, ok := finalOf(g.Layout, o.Position, o.Heading); ok && o.AGLFt < 5000 {
					name := o.Tail
					if name == "" {
						name = o.Title
					}
					feed[key{icao, end.Name}] = append(feed[key{icao, end.Name}], traffic.ApproachAircraft{Callsign: name,
						Wake: traffic.WakeFor(o.Title), DistanceToGoNM: d, GroundKts: o.GroundKts, Fixed: true})
				}
			}
		}
	}
	wx, haveWx := r.weather.Last()
	for k, list := range feed {
		sk := k.icao + " " + k.end
		s := a.seqs[sk]
		if s == nil {
			s = traffic.NewApproachSequencer(k.end, traffic.SequencerOptions{MinSpacingNM: traffic.EnrouteSeparationNM})
			a.seqs[sk] = s
		}
		if g := graph(k.icao); g != nil && haveWx {
			if _, end, ok := g.Layout.RunwayEnd(k.end); ok {
				s.SetConditions(traffic.ConditionsFrom(wx, end.Heading))
			}
		}
		seq := s.Update(now, list)
		a.absorb(r, now, k.icao, seq, byCS)
	}
	// Sequencers without arrivals left: forget them (a fresh one next time).
	for sk := range a.seqs {
		i, e, _ := strings.Cut(sk, " ")
		if _, ok := feed[key{i, e}]; !ok {
			delete(a.seqs, sk)
		}
	}
}

// absorb has our arrivals on their STAR lose their delay: speed, then a
// longer downwind; the rest in the hold, left at atcHoldRelease.
func (a *atcState) absorb(r *Runtime, now time.Time, icao string, seq []traffic.SequenceEntry, byCS map[string]*flight) {
	for _, e := range seq {
		f := byCS[e.Callsign]
		if f == nil || f.held || e.Fixed {
			continue
		}
		if h, _, holding := f.arr.Holding(); holding {
			if e.Delay <= atcHoldRelease {
				a.leaveHold(now, icao, f, h, e)
			}
			continue
		}
		if e.Delay < atcAbsorbFrom || now.Sub(a.absorbed[e.Callsign]) < atcAbsorbEvery {
			continue
		}
		ab, err := f.arr.AbsorbDelay(e.Delay)
		if errors.Is(err, traffic.ErrNotOnProcedure) || errors.Is(err, traffic.ErrHolding) {
			continue
		}
		a.absorbed[e.Callsign] = now
		if err != nil {
			a.say(now, icao, e.Callsign, fmt.Sprintf("%s: absorbing %s failed: %v", e.Callsign, e.Delay.Round(time.Second), err))
			continue
		}
		a.say(now, icao, e.Callsign, fmt.Sprintf("%s, number %d, delay %s: %s", e.Callsign, e.Number, e.Delay.Round(time.Second), ab))
		if ab.Left >= atcHoldFrom {
			a.enterHold(now, icao, f, e, ab.Left)
		}
	}
}

func (a *atcState) stack(icao string, h traffic.Hold) *traffic.HoldStack {
	k := fmt.Sprintf("%s %.3f %.3f", icao, h.Fix.Lat, h.Fix.Lon)
	if st := a.stacks[k]; st != nil {
		return st
	}
	st := &traffic.HoldStack{Hold: h, BaseFt: atcHoldBaseFt}
	a.stacks[k] = st
	return st
}

func holdName(h traffic.Hold) string {
	if h.Ident != "" {
		return h.Ident
	}
	return fmt.Sprintf("%.3f %.3f", h.Fix.Lat, h.Fix.Lon)
}

func (a *atcState) enterHold(now time.Time, icao string, f *flight, e traffic.SequenceEntry, left time.Duration) error {
	h, ok := f.arr.HoldFix(atcHoldFixNM)
	if !ok {
		return fmt.Errorf("no STAR fix %.0f NM or more out to hold at", atcHoldFixNM)
	}
	st := a.stack(icao, h)
	h = st.Hold
	alt := st.Assign(e.Callsign)
	entry, err := f.arr.EnterHold(h, alt)
	if err != nil {
		st.Release(e.Callsign)
		return err
	}
	a.say(now, icao, e.Callsign, fmt.Sprintf("%s, hold at %s, %s entry, maintain %.0f ft, expect further clearance %s",
		e.Callsign, holdName(h), entry, alt, now.Add(left).UTC().Format("1504Z")))
	return nil
}

func (a *atcState) leaveHold(now time.Time, icao string, f *flight, h traffic.Hold, e traffic.SequenceEntry) error {
	if err := f.arr.LeaveHold(); err != nil {
		return err
	}
	a.say(now, icao, e.Callsign, fmt.Sprintf("%s, leave the hold at %s, number %d, continue the arrival", e.Callsign, holdName(h), e.Number))
	byCS := map[string]*flight{}
	if t := f.ts; t != nil {
		t.tmu.Lock()
		for cs, x := range t.flights {
			byCS[cs] = x
		}
		t.tmu.Unlock()
	}
	for cs, alt := range a.stack(icao, h).Release(e.Callsign) {
		if above := byCS[cs]; above != nil && above.arr != nil && above.arr.HoldAltitude(alt) == nil {
			a.say(now, icao, cs, fmt.Sprintf("%s, descend %.0f ft, hold as published", cs, alt))
		}
	}
	return nil
}

// Sequences implements Traffic.
func (r *Runtime) Sequences(icao string) []RunwaySequence {
	icao = strings.ToUpper(strings.TrimSpace(icao))
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.traffic
	if t == nil || t.atc == nil {
		return nil
	}
	var out []RunwaySequence
	for k, s := range t.atc.seqs {
		i, end, _ := strings.Cut(k, " ")
		if icao != "" && i != icao {
			continue
		}
		out = append(out, RunwaySequence{ICAO: i, Runway: end, Conditions: s.Conditions(), Sequence: s.Sequence()})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ICAO+out[a].Runway < out[b].ICAO+out[b].Runway })
	return out
}

// Tower implements Traffic.
func (r *Runtime) Tower(icao string) []RunwayUser {
	icao = strings.ToUpper(strings.TrimSpace(icao))
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.traffic
	if t == nil || t.atc == nil {
		return nil
	}
	var out []RunwayUser
	for i, us := range t.atc.users {
		if icao == "" || i == icao {
			out = append(out, us...)
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Runway < out[b].Runway })
	return out
}

// ATCLog implements Traffic.
func (r *Runtime) ATCLog(limit int) []ATCMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.traffic
	if t == nil || t.atc == nil {
		return nil
	}
	l := t.atc.log
	if limit > 0 && len(l) > limit {
		l = l[len(l)-limit:]
	}
	return append([]ATCMessage(nil), l...)
}

// Approach implements Traffic: a controller's instruction to one of our
// arrivals in a landing sequence. It returns what was said.
func (r *Runtime) Approach(callsign, action string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.traffic
	if t == nil || t.atc == nil {
		return "", fmt.Errorf("%s: %w", callsign, ErrNotSequenced)
	}
	a := t.atc
	var s *traffic.ApproachSequencer
	var e traffic.SequenceEntry
	icao := ""
	for k, q := range a.seqs {
		for _, x := range q.Sequence() {
			if x.Callsign == callsign {
				s, e = q, x
				icao, _, _ = strings.Cut(k, " ")
			}
		}
	}
	if s == nil {
		return "", fmt.Errorf("%s: %w", callsign, ErrNotSequenced)
	}
	t.tmu.Lock()
	f := t.flights[callsign]
	t.tmu.Unlock()
	now := time.Now()
	switch action {
	case "up", "down":
		places := -1
		if action == "down" {
			places = 1
		}
		if err := s.Move(callsign, places); err != nil {
			return "", err
		}
		msg := fmt.Sprintf("%s moved %s in the sequence to %s", callsign, action, s.Runway())
		a.say(now, icao, callsign, msg)
		return msg, nil
	}
	if f == nil || f.arr == nil {
		return "", fmt.Errorf("%s: %w", callsign, ErrUnknownFlight)
	}
	before := len(a.log)
	switch action {
	case "slow":
		ab, err := f.arr.AbsorbDelay(atcSlowBy)
		if err != nil {
			return "", err
		}
		a.absorbed[callsign] = now
		a.say(now, icao, callsign, fmt.Sprintf("%s, number %d, lose a minute: %s", callsign, e.Number, ab))
	case "hold":
		if _, _, holding := f.arr.Holding(); holding {
			return "", traffic.ErrHolding
		}
		if err := a.enterHold(now, icao, f, e, max(e.Delay, 2*time.Minute)); err != nil {
			return "", err
		}
	case "release":
		h, _, holding := f.arr.Holding()
		if !holding {
			return "", traffic.ErrNotHolding
		}
		if err := a.leaveHold(now, icao, f, h, e); err != nil {
			return "", err
		}
	case "direct":
		if err := f.arr.DirectToJoin(); err != nil {
			return "", err
		}
		a.say(now, icao, callsign, fmt.Sprintf("%s, proceed direct to the final, number %d", callsign, e.Number))
	case "goaround":
		if err := f.arr.GoAround(); err != nil {
			return "", err
		}
		a.rejoin(icao, callsign)
		a.say(now, icao, callsign, callsign+", go around, I say again, go around")
	default:
		return "", fmt.Errorf("%s: unknown instruction %q (up, down, slow, hold, release, direct, goaround)", callsign, action)
	}
	var said []string
	for _, m := range a.log[before:] {
		said = append(said, m.Text)
	}
	return strings.Join(said, "; "), nil
}
