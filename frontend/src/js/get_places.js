// TODO: Better organization of this file + additional documentation
//       The flow is pretty confusing right now.
// In the meantime, here's what happens in this script:
//     * On DOM load, fetch the population density TIF and Map (id in URL),
//       then based on that information, populate foundCoords with panos
//       from the streetview API
//     * Once we have mapSettings.NumRounds panos in foundCoords, automatically
//       submit a POST request to the server with a new Challenge containing
//       those coords.  The server responds with the ID of the new Challenge,
//       which we store.  Then, enable the "Start Challenge" button.
//     * When the user presses the button, we read their Nickname from the form
//       and send it along with the saved ChallengeID as a new ChallengeResult.
//       The server again responds with the ChallengeResult's ID.  We set two
//       cookies storing the ChallengeID and ChallengeResultID, then redirect
//       to /play.

import { point, booleanPointInPolygon, bbox } from '@turf/turf';
import * as GeoTIFF from 'geotiff';

// search radius in meters - using 500 (formerly 50,000) causes more NO_RESULTS
// responses, but the API also takes much less time to fulfill the requests.
// It also means we can use StreetViewPreference.BEST without so many duplicate
// responses.
const PANO_SEARCH_RADIUS = 500;
// discard polar panos, they're usually garbage
const LAT_LIMIT = 85;
// fetchPano will query the streetview API this many times before giving up
const MAX_REQS = 100;
// getRandomConstrainedLatLng will try this many random latlngs before giving up
const MAX_LATLNG_ATTEMPTS = 1000000;

// == POPULATION DENSITY ========
// TODO: can we find another way to do population density?
//       This TIF is 6.5mb.
//       At minimum, cache it.
export async function loadGeoTIF(loc) {
    const response = await fetch("/public/assets/nasa_pop_data.tif");
    const arrayBuffer = await response.arrayBuffer();
    return await GeoTIFF.fromArrayBuffer(arrayBuffer);
}

// get normalized (0.0 - 1.0) population density at lat, lng
export async function getLocationPopulation(popTIF, lat, lng) {
    const delta = 0.01;
    let value = await popTIF.readRasters({
        bbox: [lng, lat, lng + 10*delta, lat + 10*delta],
        resX: delta,
        resY: delta,
    });
    let actualValue = value[0][0];
    // 255 means ocean
    if (actualValue == 255) {
        actualValue = 0;
    }
    return actualValue / 255;
}

// == GET PANOS ========
export async function fetchPanos(svService, settings, popTIF, incrNumReqsCallback = () => {}) {
    const promises = [];
    for (let i = 0; i < settings.NumRounds; i++) {
        promises.push(fetchPano(svService, settings, popTIF, incrNumReqsCallback));
    }
    let foundLatLngs = await Promise.all(promises);
    return foundLatLngs;
}

export async function fetchPano(svService, settings, popTIF, incrNumReqsCallback) {
    let source = settings.Source == 1 ? google.maps.StreetViewSource.OUTDOOR : google.maps.StreetViewSource.DEFAULT;
    let randomLatLng;
    let foundLatLng = null;
    for (let iters = 0; iters < MAX_REQS; iters++) {
        randomLatLng = await getRandomConstrainedLatLng(settings.Polygon, popTIF, settings.MinDensity, settings.MaxDensity);
        if (!randomLatLng) {
            // couldn't find a good latlng (one meeting pop density and polygon requirements)
            console.log("Maximum number of latlng generation attempts exceeded.");
            return null;
        }
        foundLatLng = await new Promise((resolve, reject) => {
            svService.getPanorama({
                location: randomLatLng,
                // NEAREST or BEST.  BEST seems to give more actual streetview results (rather
                // than third party photospheres) so I'm going with that.
                preference: google.maps.StreetViewPreference.BEST,
                radius: PANO_SEARCH_RADIUS,
                source: source,
            }, (result, status) => {resolve(handlePanoResponse(result, status));});
        });
        incrNumReqsCallback(foundLatLng);
        if (foundLatLng) {
            return foundLatLng;
        }
    }
    
    function handlePanoResponse(result, status) {
        if (status == google.maps.StreetViewStatus.OK && resultPanoIsGood(result, settings)) {
            return {
                "latLng": result.location.latLng,
                "panoID": result.location.pano
            };
        }
    }
    console.log("Maximum number of StreetView API requests exceeded.");
    return null;
}

// returns whether result (pano) meets the requirements of mapInfo
export function resultPanoIsGood(result, settings) {
    if (result.location.latLng.lat() > LAT_LIMIT || result.location.latLng.lat() < -1 * LAT_LIMIT) {return false;}

    if (settings.Copyright == 1 && !result.copyright.includes("Google")) {
        return false;
    }
    if (settings.Copyright == 2 && result.copyright.includes("Google")) {
        return false;
    }

    if (settings.Connectedness == 1 && result.links.length == 0) {
        return false;
    }
    if (settings.Connectedness == 2 && result.links.length > 0) {
        return false;
    }

    let locationTurfPoint = point([result.location.latLng.lng(), result.location.latLng.lat()]);
    if (settings.Polygon != null && !booleanPointInPolygon(locationTurfPoint, settings.Polygon)) {
        return false;
    }

    // TODO: duplicate checking that doesn't rely on globals
    //       and doesn't have a race condition

    return true;
}

// get a random google.maps.LatLng within the specified polygon and with
// a population density in the specified range
export async function getRandomConstrainedLatLng(polygon, popTIF, minDensity, maxDensity) {
    // TODO: function assignment as control flow is heinous
    let getRandomLngLatInBounds;
    let pointInPolygon;
    if (polygon == null) {
        getRandomLngLatInBounds = getRandomLngLat;
        pointInPolygon = (_) => true;
    } else {
        let bounds = bbox(polygon);
        getRandomLngLatInBounds = function() {
            let randomLng = (Math.random() * (bounds[2] - bounds[0]) + bounds[0]);
            let randomLat = (Math.random() * (bounds[3] - bounds[1]) + bounds[1]);
            return [randomLng, randomLat];
        }
        pointInPolygon = function(lnglat) {
            return booleanPointInPolygon(point(lnglat), polygon);
        }
    }

    async function popDensityInLimits(lnglat) {
        let density = (await getLocationPopulation(popTIF, lnglat[1], lnglat[0])) * 100;
        return density <= maxDensity && density >= minDensity;
    }
    
    let attempts = 0;
    let lnglat;
    do {
        lnglat = getRandomLngLatInBounds();
        if (attempts > MAX_LATLNG_ATTEMPTS) {
            return null;
        }
        attempts++;
    } while (!pointInPolygon(lnglat) || !(await popDensityInLimits(lnglat)));
    return new google.maps.LatLng(lnglat[1], lnglat[0]);
}

// get a random google.maps.LatLng, anywhere
export function getRandomLngLat() {
    let randomLng = (Math.random() * 360 - 180);
    let randomLat = (Math.random() * 180 - 90);
    return [randomLng, randomLat];
}
