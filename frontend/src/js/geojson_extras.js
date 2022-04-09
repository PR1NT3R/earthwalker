// functions for interacting with geoJSON
// mainly helpers for MapSettings
import { multiPolygon } from '@turf/turf';


const NOMINATIM_URL = (locStringEncoded) => `https://nominatim.openstreetmap.org/search?q=${locStringEncoded}&polygon_geojson=1&limit=5&polygon_threshold=0.005&format=json`;

// combine MultiPolygons into a single MultiPolygon
// falsey items are skipped
export function combinePolygons(polygons) {
    let polygon = null;
    for (let i = 0; i < polygons.length; i++) {
        if (polygons[i]) {
            if (polygon) {
                polygon.geometry.coordinates.push(
                    ...polygons[i].geometry.coordinates);
            } else {
                // deep copy TODO: awk
                polygon = JSON.parse(JSON.stringify(polygons[i]));
            }
        }
    }
    return polygon
}

export async function polygonFromLocString(locString) {
    if (!locString) {
        return null;
    }

    let data;
    let response = await fetch(
        NOMINATIM_URL(encodeURI(locString)),
        {cache: "force-cache"}); // cache aggressively
    if (response.ok) {
        data = await response.json();
    } else {
        return null;
    }

    let polygon = geojsonFromNominatim(data)
    if (polygon) {
        return polygon;
    } else {
        return null;
    }
}

// given Nominatim results, takes the most significant one with a polygon or
// multipolygon and returns it as a turf.multiPolygon
function geojsonFromNominatim(data) {
    for (let i = 0; i < data.length; i++) {
        let type = data[i].geojson.type.toLowerCase();
        if (type === "multipolygon") {
            return multiPolygon(data[i].geojson.coordinates);
        } else if (type === "polygon") {
            return multiPolygon([data[i].geojson.coordinates]);
        }
    }
    console.log("No matching polygon!");
    return null;
}