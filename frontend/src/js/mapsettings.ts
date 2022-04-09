import { polygonFromLocString, combinePolygons } from "./geojson_extras";
import { area } from '@turf/turf';

// MapSettings handles the temporary representation of a map held by
// the frontend during Map editing.  Note that right now, most frontend pages
// interact directly with the JSON format provided by the backend (a "Map")
// rather than using this class.
export type MapSettings = {
    // properties expected by the backend (for actual Map creation)
    name: string;
    polygon: any; // TODO: type?
    area: number;
    numRounds: number;
    timeLimit: number;
    graceDistance: number;
    minDensity: number;
    maxDensity: number;
    connectedness: number;
    copyright: number;
    source: number;
    showLabels: boolean;

    // properties expected by the frontend (for editing)
    timeLimitMinutes?: number;
    timeLimitSeconds?: number;
    drawnPolygons: any[]; // MultiPolygon TODO: type?
    locStrings?: string[];
    stringPolygons: {};
};


export async function updateStringPolygons(
    settings: MapSettings,
    locStrings: string[]
) {
    if (!settings.stringPolygons) {
        settings.stringPolygons = {}
    }
    let newPolys = {};
    for (const locString of locStrings) {
        if (settings.stringPolygons.hasOwnProperty(locString)) {
            newPolys[locString] = settings.stringPolygons[locString];
        } else {
            newPolys[locString] = await polygonFromLocString(locString);
        }
    }
    settings.stringPolygons = newPolys;
}


// update derived properties from their editable counterparts
async function updateDerivedProps(settings: MapSettings) {
    await updateStringPolygons(settings, settings.locStrings); // TODO: awk
    updatePolygon(settings);
    settings.timeLimit = settings.timeLimitMinutes * 60 + settings.timeLimitSeconds;
}


// update editable props from their back-end counterparts
function updateEditableProps(settings: MapSettings) {
    settings.timeLimitSeconds = settings.timeLimit % 60;
    settings.timeLimitMinutes = (settings.timeLimit - settings.timeLimitSeconds) / 60;
}


export function mapSettingsToJson(settings: MapSettings) {
    return JSON.stringify({
            mapName: settings.name,
            numRounds: settings.numRounds,
            graceDistance: settings.graceDistance,
            minDensity: settings.minDensity,
            maxDensity: settings.maxDensity,
            connectedness: settings.connectedness,
            copyright: settings.copyright,
            source: settings.source,
            showLabels: settings.showLabels,
            timeLimitSeconds: settings.timeLimitSeconds,
            timeLimitMinutes: settings.timeLimitMinutes,
            locStrings: Object.keys(settings.stringPolygons),
            drawnPolygons: settings.drawnPolygons
        },
        null,
        2);
}


export async function mapSettingsFromJson(mapJSON): Promise<MapSettings> {
    let settings: MapSettings = JSON.parse(mapJSON);
    await updateDerivedProps(settings);
    return settings;
}


// produce a Map object as expected by the /api/maps backend endpoint
// (see domain.go)
export function mapSettingsToMap(settings: MapSettings) {
    updateDerivedProps(settings);
    return {
        Name: settings.name,
        Polygon: settings.polygon,
        Area: settings.area,
        NumRounds: settings.numRounds,
        TimeLimit: settings.timeLimit,
        GraceDistance: settings.graceDistance,
        MinDensity: settings.minDensity,
        MaxDensity: settings.maxDensity,
        Connectedness: settings.connectedness,
        Copyright: settings.copyright,
        Source: settings.source,
        ShowLabels: settings.showLabels
    };
}


export function mapSettingsFromMap() {
    
}


// TODO: don't completely reset settings.polygon every update
export function updatePolygon(settings: MapSettings) {
    let polygons = [];
    settings.drawnPolygons.forEach(poly => {
        polygons.push(poly);
    });
    Object.values(settings.stringPolygons).forEach(poly => {
        polygons.push(poly);
    });
    settings.polygon = combinePolygons(polygons);

    if (settings.polygon) {
        settings.area = area(settings.polygon);
    } else {
        settings.area = 0;
    }
}
