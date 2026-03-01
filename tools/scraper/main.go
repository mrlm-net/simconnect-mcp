package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mrlm-net/simconnect-mcp/internal/corpus"
)

// ── URL catalogs ──────────────────────────────────────────────────────────────
//
// These are the canonical docs.flightsimulator.com pages for each corpus type
// and SDK version. Each entry maps to a scrape+parse operation. Only the pages
// listed here are fetched — the scraper does NOT crawl links.
//
// Pages that return 404 are silently skipped (the scraper logs a warning and
// continues), so it is safe to list candidate pages for both versions even if
// a page only exists in one of them.

type pageSpec struct {
	url      string
	category string
}

// ── 2020 base URLs ────────────────────────────────────────────────────────────
const (
	base2020SimVars = "https://docs.flightsimulator.com/html/Programming_Tools/SimVars/"
	base2020Events  = "https://docs.flightsimulator.com/html/Programming_Tools/Event_IDs/"
	base2020SimConn = "https://docs.flightsimulator.com/html/Programming_Tools/SimConnect/"
)

// ── 2024 base URLs ────────────────────────────────────────────────────────────
const (
	base2024Root    = "https://docs.flightsimulator.com/msfs2024/html/6_Programming_APIs/"
	base2024SimVars = "https://docs.flightsimulator.com/msfs2024/html/6_Programming_APIs/SimVars/"
	base2024Events  = "https://docs.flightsimulator.com/msfs2024/html/6_Programming_APIs/Key_Events/"
	base2024SimConn = "https://docs.flightsimulator.com/msfs2024/html/6_Programming_APIs/SimConnect/"
)

var simvarPages2020 = []pageSpec{
	// ── Aircraft SimVars ──────────────────────────────────────────────────────
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_AutopilotAssistant_Variables.htm", category: "Autopilot and Assistant"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_Brake_Landing_Gear_Variables.htm", category: "Brakes and Landing Gear"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_Control_Variables.htm", category: "Controls"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_Electrics_Variables.htm", category: "Electrics"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_Engine_Variables.htm", category: "Engine"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_FlightModel_Variables.htm", category: "Flight Model"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_Fuel_Variables.htm", category: "Fuel"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_Misc_Variables.htm", category: "Miscellaneous"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_RadioNavigation_Variables.htm", category: "Radio Navigation"},
	{url: base2020SimVars + "Aircraft_SimVars/Aircraft_System_Variables.htm", category: "System"},
	// ── Non-aircraft SimVars ──────────────────────────────────────────────────
	{url: base2020SimVars + "Helicopter_Variables.htm", category: "Helicopter"},
	{url: base2020SimVars + "Camera_Variables.htm", category: "Camera"},
	{url: base2020SimVars + "Services_Variables.htm", category: "Services"},
	{url: base2020SimVars + "Miscellaneous_Variables.htm", category: "Miscellaneous Variables"},
	{url: base2020SimVars + "RTPC_And_Simulation_Variables.htm", category: "RTPC"},
	// ── Environment variables ─────────────────────────────────────────────────
	{url: "https://docs.flightsimulator.com/html/Programming_Tools/Environment_Variables.htm", category: "Environment"},
}

var simvarPages2024 = []pageSpec{
	// ── Aircraft SimVars ──────────────────────────────────────────────────────
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_AutopilotAssistant_Variables.htm", category: "Autopilot and Assistant"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_Brake_Landing_Gear_Variables.htm", category: "Brakes and Landing Gear"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_Control_Variables.htm", category: "Controls"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_Electrics_Variables.htm", category: "Electrics"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_Engine_Variables.htm", category: "Engine"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_FlightModel_Variables.htm", category: "Flight Model"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_Fuel_Variables.htm", category: "Fuel"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_Misc_Variables.htm", category: "Miscellaneous"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_RadioNavigation_Variables.htm", category: "Radio Navigation"},
	{url: base2024SimVars + "Aircraft_SimVars/Aircraft_System_Variables.htm", category: "System"},
	// ── Non-aircraft SimVars ──────────────────────────────────────────────────
	{url: base2024SimVars + "Helicopter_Variables.htm", category: "Helicopter"},
	{url: base2024SimVars + "Camera_Variables.htm", category: "Camera"},
	{url: base2024SimVars + "Balloon_Variables.htm", category: "Balloon"},
	{url: base2024SimVars + "Services_Variables.htm", category: "Services"},
	{url: base2024SimVars + "Miscellaneous_Variables.htm", category: "Miscellaneous Variables"},
	// ── Environment variables ─────────────────────────────────────────────────
	{url: base2024Root + "Environment_Variables.htm", category: "Environment"},
}

var eventPages2020 = []pageSpec{
	{url: base2020Events + "Aircraft_Autopilot_Flight_Assist_Events.htm"},
	{url: base2020Events + "Aircraft_Electrical_Events.htm"},
	{url: base2020Events + "Aircraft_Engine_Events.htm"},
	{url: base2020Events + "Aircraft_Flight_Control_Events.htm"},
	{url: base2020Events + "Aircraft_Fuel_System_Events.htm"},
	{url: base2020Events + "Aircraft_Instrumentation_Events.htm"},
	{url: base2020Events + "Aircraft_Misc_Events.htm"},
	{url: base2020Events + "Aircraft_Radio_Navigation_Events.htm"},
	{url: base2020Events + "Helicopter_Specific_Events.htm"},
	{url: base2020Events + "Miscellaneous_Events.htm"},
	{url: base2020Events + "View_Camera_Events.htm"},
}

var eventPages2024 = []pageSpec{
	{url: base2024Events + "Aircraft_Autopilot_Flight_Assist_Events.htm"},
	{url: base2024Events + "Aircraft_Electrical_Events.htm"},
	{url: base2024Events + "Aircraft_Engine_Events.htm"},
	{url: base2024Events + "Aircraft_Flight_Control_Events.htm"},
	{url: base2024Events + "Aircraft_Fuel_System_Events.htm"},
	{url: base2024Events + "Aircraft_General_Systems.htm"},
	{url: base2024Events + "Aircraft_Instrumentation_Events.htm"},
	{url: base2024Events + "Aircraft_Misc_Events.htm"},
	{url: base2024Events + "Aircraft_Radio_Navigation_Events.htm"},
	{url: base2024Events + "Helicopter_Specific_Events.htm"},
	{url: base2024Events + "Balloon_Airship_Events.htm"},
	{url: base2024Events + "Miscellaneous_Events.htm"},
	{url: base2024Events + "View_Camera_Events.htm"},
}

// functionPages lists every SimConnect API function page. Pages are scraped
// from the 2020 docs by default; 2024-only functions use base2024SimConn.
// Unknown or 404 pages are silently skipped by the scraper.
var functionPages = []string{
	// ── General ───────────────────────────────────────────────────────────────
	base2020SimConn + "API_Reference/General/SimConnect_Open.htm",
	base2020SimConn + "API_Reference/General/SimConnect_Close.htm",
	base2020SimConn + "API_Reference/General/SimConnect_CallDispatch.htm",
	base2020SimConn + "API_Reference/General/SimConnect_GetNextDispatch.htm",
	base2020SimConn + "API_Reference/General/SimConnect_RequestSystemState.htm",
	base2020SimConn + "API_Reference/General/SimConnect_SetNotificationGroupPriority.htm",
	base2020SimConn + "API_Reference/General/SimConnect_ExecuteAction.htm",

	// ── Events and Data ───────────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_MapClientEventToSimEvent.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_TransmitClientEvent.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_TransmitClientEvent_EX1.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_AddClientEventToNotificationGroup.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_RemoveClientEvent.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_SubscribeToSystemEvent.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_UnsubscribeFromSystemEvent.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_SetSystemEventState.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_RequestDataOnSimObject.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_RequestDataOnSimObjectType.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_AddToDataDefinition.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_ClearDataDefinition.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_SetDataOnSimObject.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_MapClientDataNameToID.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_CreateClientData.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_AddToClientDataDefinition.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_ClearClientDataDefinition.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_RequestClientData.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_SetClientData.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_RequestNotificationGroup.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_ClearNotificationGroup.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_ClearInputGroup.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_SetInputGroupPriority.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_SetInputGroupState.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_RemoveInputEvent.htm",
	base2020SimConn + "API_Reference/Events_And_Data/SimConnect_RequestReservedKey.htm",

	// ── Flights ───────────────────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Flights/SimConnect_FlightLoad.htm",
	base2020SimConn + "API_Reference/Flights/SimConnect_FlightSave.htm",
	base2020SimConn + "API_Reference/Flights/SimConnect_FlightPlanLoad.htm",

	// ── Debug ─────────────────────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Debug/SimConnect_GetLastSentPacketID.htm",
	base2020SimConn + "API_Reference/Debug/SimConnect_RequestResponseTimes.htm",
	base2020SimConn + "API_Reference/Debug/SimConnect_InsertString.htm",
	base2020SimConn + "API_Reference/Debug/SimConnect_RetrieveString.htm",

	// ── Facilities ────────────────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Facilities/SimConnect_AddToFacilityDefinition.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_AddFacilityDataDefinitionFilter.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_ClearAllFacilityDataDefinitionFilters.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_RequestFacilitesList.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_RequestFacilitiesList_EX1.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_RequestFacilityData.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_RequestFacilityData_EX1.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_RequestJetwayData.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_SubscribeToFacilities.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_SubscribeToFacilities_EX1.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_UnsubscribeToFacilities.htm",
	base2020SimConn + "API_Reference/Facilities/SimConnect_UnsubscribeToFacilities_EX1.htm",

	// ── AI Objects (both versions) ───────────────────────────────────────────
	base2020SimConn + "API_Reference/AI_Objects/SimConnect_AICreateEnrouteATCAircraft.htm",
	base2020SimConn + "API_Reference/AI_Objects/SimConnect_AICreateNonATCAircraft.htm",
	base2020SimConn + "API_Reference/AI_Objects/SimConnect_AICreateParkedATCAircraft.htm",
	base2020SimConn + "API_Reference/AI_Objects/SimConnect_AICreateSimulatedObject.htm",
	base2020SimConn + "API_Reference/AI_Objects/SimConnect_AIReleaseControl.htm",
	base2020SimConn + "API_Reference/AI_Objects/SimConnect_AIRemoveObject.htm",
	base2020SimConn + "API_Reference/AI_Objects/SimConnect_AISetAircraftFlightPlan.htm",
	// ── 2024-only: AI Objects EX1 variants + SimObjectsAndLiveries ───────────
	base2024SimConn + "API_Reference/AI_Objects/SimConnect_AICreateEnrouteATCAircraft_EX1.htm",
	base2024SimConn + "API_Reference/AI_Objects/SimConnect_AICreateNonATCAircraft_EX1.htm",
	base2024SimConn + "API_Reference/AI_Objects/SimConnect_AICreateParkedATCAircraft_EX1.htm",
	base2024SimConn + "API_Reference/AI_Objects/SimConnect_AICreateSimulatedObject_EX1.htm",
	base2024SimConn + "API_Reference/AI_Objects/SimConnect_EnumerateSimObjectsAndLiveries.htm",

	// ── 2024-only: Input Events (folder name has no underscore: InputEvents/) ──
	base2024SimConn + "API_Reference/InputEvents/SimConnect_EnumerateControllers.htm",
	base2024SimConn + "API_Reference/InputEvents/SimConnect_EnumerateInputEvents.htm",
	base2024SimConn + "API_Reference/InputEvents/SimConnect_EnumerateInputEventParams.htm",
	base2024SimConn + "API_Reference/InputEvents/SimConnect_GetInputEvent.htm",
	base2024SimConn + "API_Reference/InputEvents/SimConnect_MapInputEventToClientEvent.htm",
	base2024SimConn + "API_Reference/InputEvents/SimConnect_MapInputEventToClientEvent_EX1.htm",
	base2024SimConn + "API_Reference/InputEvents/SimConnect_SetInputEvent.htm",
	base2024SimConn + "API_Reference/InputEvents/SimConnect_SubscribeInputEvent.htm",
	base2024SimConn + "API_Reference/InputEvents/SimConnect_UnsubscribeInputEvent.htm",

	// ── 2024-only: Flow API (lives under Events_And_Data/, not a separate folder) ─
	base2024SimConn + "API_Reference/Events_And_Data/SimConnect_SubscribeToFlowEvent.htm",
	base2024SimConn + "API_Reference/Events_And_Data/SimConnect_UnsubscribeToFlowEvent.htm",
}

// structurePages lists every SimConnect structure and enumeration page.
// Unknown or 404 pages are silently skipped.
var structurePages = []string{
	// ── Base structures ───────────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_OPEN.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_QUIT.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EXCEPTION.htm",

	// ── SimObject data structures ─────────────────────────────────────────────
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_SIMOBJECT_DATA.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_SIMOBJECT_DATA_BYTYPE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_ASSIGNED_OBJECT_ID.htm",

	// ── Event structures ──────────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_FILENAME.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_FRAME.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_OBJECT_ADDREMOVE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_RACE_END.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_RACE_LAP.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_MULTIPLAYER_CLIENT_STARTED.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_MULTIPLAYER_SERVER_STARTED.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_MULTIPLAYER_SESSION_ENDED.htm",

	// ── Client data structures ────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_CLIENT_DATA.htm",

	// ── System state ──────────────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_SYSTEM_STATE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_RESERVED_KEY.htm",

	// ── Facilities structures ─────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_FACILITIES_LIST.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_AIRPORT_LIST.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_NDB_LIST.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_VOR_LIST.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_WAYPOINT_LIST.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_LIST_TEMPLATE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_FACILITY_DATA.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_FACILITY_DATA_END.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_FACILITY_MINIMAL_LIST.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_JETWAY_DATA.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_FACILITY_AIRPORT.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_FACILITY_NDB.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_FACILITY_VOR.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_FACILITY_WAYPOINT.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_FACILITY_MINIMAL.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_ICAO.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_JETWAY_DATA.htm",

	// ── Data type structures ──────────────────────────────────────────────────
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_INITPOSITION.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_LATLONALT.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_PBH.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_XYZ.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_WAYPOINT.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_MARKERSTATE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATA_RACE_RESULT.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_VERSION_BASE_TYPE.htm",

	// ── Enumerations (listed alongside structures in the API reference) ──────
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_CLIENT_DATA_PERIOD.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_DATATYPE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_FACILITY_DATA_TYPE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_FACILITY_LIST_TYPE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_INPUT_EVENT_TYPE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_MISSION_END.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_PERIOD.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_ID.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_SIMOBJECT_TYPE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_STATE.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_TEXT_RESULT.htm",
	base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_WAYPOINT_FLAGS.htm",

	// ── 2024-only structures (Input Events, EX1 variants) ─────────────────────
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_EVENT_EX1.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_ENUMERATE_INPUT_EVENTS.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_ENUMERATE_INPUT_EVENT_PARAMS.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_GET_INPUT_EVENT.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_SUBSCRIBE_INPUT_EVENT.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_CONTROLLERS_LIST.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_INPUT_EVENT_DESCRIPTOR.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_CONTROLLER_ITEM.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_ENUMERATE_SIMOBJECT_LIVERY.htm",
	base2024SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_RECV_ENUMERATE_SIMOBJECT_AND_LIVERY_LIST.htm",
}

var errorCodePage = base2020SimConn + "API_Reference/Structures_And_Enumerations/SIMCONNECT_EXCEPTION.htm"

// ── scrapeResult bundles a file write outcome ─────────────────────────────────

type scrapeResult struct {
	filename string
	itemType string
	count    int
	err      error
}

func main() {
	outDir := flag.String("out", "internal/corpus/assets/", "output directory for JSON files")
	version := flag.String("version", "both", "SDK version to scrape: 2020, 2024, or both")
	dryRun := flag.Bool("dry-run", false, "parse only, do not write output files")
	delay := flag.Duration("delay", time.Second, "minimum delay between HTTP requests")
	flag.Parse()

	if *version != "2020" && *version != "2024" && *version != "both" {
		log.Fatalf("-version must be 2020, 2024, or both; got %q", *version)
	}

	start := time.Now()
	s := NewScraper(*delay)

	var results []scrapeResult
	var failCount int

	// ── SimVars ───────────────────────────────────────────────────────────────

	if *version == "2020" || *version == "both" {
		r := scrapeSimVars(s, simvarPages2020, "2020", *outDir, *dryRun)
		results = append(results, r)
		if r.err != nil {
			failCount++
		}
	}
	if *version == "2024" || *version == "both" {
		r := scrapeSimVars(s, simvarPages2024, "2024", *outDir, *dryRun)
		results = append(results, r)
		if r.err != nil {
			failCount++
		}
	}

	// ── Events ────────────────────────────────────────────────────────────────

	if *version == "2020" || *version == "both" {
		r := scrapeEvents(s, eventPages2020, "2020", *outDir, *dryRun)
		results = append(results, r)
		if r.err != nil {
			failCount++
		}
	}
	if *version == "2024" || *version == "both" {
		r := scrapeEvents(s, eventPages2024, "2024", *outDir, *dryRun)
		results = append(results, r)
		if r.err != nil {
			failCount++
		}
	}

	// ── Functions (version-agnostic) ──────────────────────────────────────────

	r := scrapeFunctions(s, functionPages, *outDir, *dryRun)
	results = append(results, r)
	if r.err != nil {
		failCount++
	}

	// ── Structures (version-agnostic) ─────────────────────────────────────────

	r = scrapeStructures(s, structurePages, *outDir, *dryRun)
	results = append(results, r)
	if r.err != nil {
		failCount++
	}

	// ── Error Codes (version-agnostic) ───────────────────────────────────────

	r = scrapeErrorCodes(s, errorCodePage, *outDir, *dryRun)
	results = append(results, r)
	if r.err != nil {
		failCount++
	}

	// ── Summary ───────────────────────────────────────────────────────────────

	elapsed := time.Since(start)
	fmt.Printf("\nScrape summary (%s)\n", elapsed.Round(time.Millisecond))
	fmt.Printf("%-30s  %8s  %8s\n", "File", "Items", "Status")
	fmt.Printf("%s\n", repeatStr("-", 52))

	filesWritten := 0
	for _, res := range results {
		status := "ok"
		if res.err != nil {
			status = fmt.Sprintf("FAIL: %v", res.err)
		} else if !*dryRun {
			filesWritten++
		}
		fmt.Printf("%-30s  %8d  %s\n", res.filename, res.count, status)
	}

	if *dryRun {
		fmt.Println("\n[dry-run] No files written.")
	} else {
		fmt.Printf("\nFiles written: %d  Failures: %d\n", filesWritten, failCount)
	}

	if failCount > 0 {
		os.Exit(1)
	}
}

// ── Per-type scrape functions ─────────────────────────────────────────────────

func scrapeSimVars(s *Scraper, pages []pageSpec, ver, outDir string, dryRun bool) scrapeResult {
	filename := fmt.Sprintf("simvars_%s.json", ver)
	var allVars []corpus.SimVar

	for _, p := range pages {
		htmlBytes, err := s.FetchPage(p.url)
		if err != nil {
			log.Printf("simvars %s: fetch %s: %v", ver, p.url, err)
			continue
		}
		vars, err := ParseSimVarPage(htmlBytes, ver, p.url)
		if err != nil {
			log.Printf("simvars %s: parse %s: %v", ver, p.url, err)
			continue
		}
		allVars = append(allVars, vars...)
	}

	if len(allVars) == 0 {
		return scrapeResult{filename: filename, itemType: "simvar", err: fmt.Errorf("no simvars scraped for version %s", ver)}
	}

	c := buildCorpus(ver)
	c.SimVars = allVars

	err := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "simvar", count: len(allVars), err: err}
}

func scrapeEvents(s *Scraper, pages []pageSpec, ver, outDir string, dryRun bool) scrapeResult {
	filename := fmt.Sprintf("events_%s.json", ver)
	var allEvents []corpus.Event

	for _, p := range pages {
		htmlBytes, err := s.FetchPage(p.url)
		if err != nil {
			log.Printf("events %s: fetch %s: %v", ver, p.url, err)
			continue
		}
		events, err := ParseEventPage(htmlBytes, ver, p.url)
		if err != nil {
			log.Printf("events %s: parse %s: %v", ver, p.url, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	if len(allEvents) == 0 {
		return scrapeResult{filename: filename, itemType: "event", err: fmt.Errorf("no events scraped for version %s", ver)}
	}

	c := buildCorpus(ver)
	c.Events = allEvents

	err := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "event", count: len(allEvents), err: err}
}

func scrapeFunctions(s *Scraper, pages []string, outDir string, dryRun bool) scrapeResult {
	filename := "functions.json"
	var allFuncs []corpus.Function

	for _, u := range pages {
		htmlBytes, err := s.FetchPage(u)
		if err != nil {
			log.Printf("functions: fetch %s: %v", u, err)
			continue
		}
		fns, err := ParseFunctionPage(htmlBytes, "", u)
		if err != nil {
			log.Printf("functions: parse %s: %v", u, err)
			continue
		}
		allFuncs = append(allFuncs, fns...)
	}

	if len(allFuncs) == 0 {
		return scrapeResult{filename: filename, itemType: "function", err: fmt.Errorf("no functions scraped")}
	}

	c := buildCorpus("")
	c.Functions = allFuncs

	err := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "function", count: len(allFuncs), err: err}
}

func scrapeStructures(s *Scraper, pages []string, outDir string, dryRun bool) scrapeResult {
	filename := "structures.json"
	var allStructs []corpus.Structure

	for _, u := range pages {
		htmlBytes, err := s.FetchPage(u)
		if err != nil {
			log.Printf("structures: fetch %s: %v", u, err)
			continue
		}
		structs, err := ParseStructurePage(htmlBytes, "", u)
		if err != nil {
			log.Printf("structures: parse %s: %v", u, err)
			continue
		}
		allStructs = append(allStructs, structs...)
	}

	if len(allStructs) == 0 {
		return scrapeResult{filename: filename, itemType: "structure", err: fmt.Errorf("no structures scraped")}
	}

	c := buildCorpus("")
	c.Structures = allStructs

	err := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "structure", count: len(allStructs), err: err}
}

func scrapeErrorCodes(s *Scraper, pageURL, outDir string, dryRun bool) scrapeResult {
	filename := "error_codes.json"

	htmlBytes, err := s.FetchPage(pageURL)
	if err != nil {
		return scrapeResult{filename: filename, itemType: "errorcode", err: fmt.Errorf("fetch %s: %w", pageURL, err)}
	}

	codes, err := ParseErrorCodePage(htmlBytes, "", pageURL)
	if err != nil {
		return scrapeResult{filename: filename, itemType: "errorcode", err: err}
	}

	if len(codes) == 0 {
		return scrapeResult{filename: filename, itemType: "errorcode", err: fmt.Errorf("no error codes scraped")}
	}

	c := buildCorpus("")
	c.ErrorCodes = codes

	writeErr := writeCorpusFile(c, outDir, filename, dryRun)
	return scrapeResult{filename: filename, itemType: "errorcode", count: len(codes), err: writeErr}
}

// ── Output helpers ────────────────────────────────────────────────────────────

// buildCorpus creates a baseline Corpus with ScrapedAt set to now and
// SDKVersion set to ver (may be empty for version-agnostic types).
func buildCorpus(sdkVersion string) corpus.Corpus {
	return corpus.Corpus{
		ScrapedAt:  time.Now().UTC(),
		SDKVersion: sdkVersion,
		SimVars:    []corpus.SimVar{},
		Events:     []corpus.Event{},
		Functions:  []corpus.Function{},
		Structures: []corpus.Structure{},
		ErrorCodes: []corpus.ErrorCode{},
	}
}

// writeCorpusFile serialises c to JSON and writes it atomically to outDir/filename.
// When dryRun is true the file is not written and the function returns nil.
//
// Atomic write strategy:
//  1. Write to a temp file in the same directory.
//  2. Rename the temp file over the destination — this is atomic on POSIX and
//     best-effort on Windows (rename over existing is atomic on NTFS).
//
// If an existing file is present and the write fails, the existing file is
// preserved (the temp file is removed and the error is returned).
func writeCorpusFile(c corpus.Corpus, outDir, filename string, dryRun bool) error {
	if dryRun {
		return nil
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filename, err)
	}

	dest := filepath.Join(outDir, filename)
	tmp := dest + ".tmp"

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp for %s: %w", filename, err)
	}

	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, dest, err)
	}

	return nil
}

// repeatStr returns a string consisting of s repeated n times.
func repeatStr(s string, n int) string {
	result := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		result = append(result, s...)
	}
	return string(result)
}
