<script>
    // TODO: svelteify this file
    // TODO: this file is too large/unorganized
    import { onMount } from 'svelte';
    import Tags from 'svelte-tags-input';

    import { multiPolygon } from '@turf/turf';
    import L from 'leaflet';
    import "../../node_modules/leaflet/dist/leaflet.css";
    import "leaflet-lasso";

    import { loc, ewapi, globalMap } from '../js/stores.js';
    import {
        updatePolygon as updateSettingsPolygon,
        mapSettingsToJson,
        mapSettingsFromJson,
        mapSettingsToMap,
        mapSettingsFromMap,
        updateStringPolygons
    } from '../js/mapsettings.ts';
    import { getURLParam } from '../js/earthwalker';

    // TODO: incorporate globalMap from store?
    let mapSettings = {
        name: "",
        polygon: null,
        area: 0,
        numRounds: 5,
        timeLimit: 0,
        graceDistance: 10,
        minDensity: 15,
        maxDensity: 100,
        connectedness: 1,
        copyright: 0,
        source: 1,
        showLabels: true,
        timeLimitMinutes: 0,
        timeLimitSeconds: 0,
        drawnPolygons: [],
        stringPolygons: {}
    };

    let mapJson = "";
    
    let lasso;
    let previewMap;
    let previewPolyGroup;
    let advancedHidden = true;
    let submitDisabled = false;

    onMount(async () => {
        submitDisabled = true;
        let mapid = getURLParam("mapid");
        if (mapid) { //} && (!$globalMap || $globalMap.MapID !== mapid)) {
            $globalMap = await $ewapi.getMap(mapid);
            console.log($globalMap)
        }

        previewMap = L.map("bounds-map", {center: [0, 0], zoom: 1});
        let tileServer = (await $ewapi.getTileServer()).tileserver;
        L.tileLayer(tileServer, {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OSM</a> contributors, <a href="https://wikitech.wikimedia.org/wiki/Wikitech:Cloud_Services_Terms_of_use">Wikimedia Cloud Services</a>'
        }).addTo(previewMap);
        previewPolyGroup = L.layerGroup().addTo(previewMap);
        lasso = L.lasso(previewMap);
        previewMap.on('lasso.finished', handleLassoFinish);
        submitDisabled = false;
    });

    // collates createmap form data into a JSON object, 
    // then sends a newmap request to the server
    function handleFormSubmit() {
        if (submitDisabled) {
            return;
        }

        // TODO: evaluate challenge generation (to make sure mapSettings aren't so
        //       specific that it takes a huge number of API requests to find good
        //       panos)
        // send new map to server
        $ewapi.postMap(mapSettingsToMap(mapSettings))
            .then( (response) => {
                if (response && response.MapID) {
                    $globalMap = response; // TODO: change to a MapSettings?
                    $loc = "/createchallenge?mapid="+response.MapID;
                } else {
                    alert("Failed to submit map?!");
                }
            });
    }

    // TODO: consider moving lasso handling to another file
    function handleLassoFinish(event) {
        // Polygons with less than three points do not have an area
        if(event.latLngs.length > 2){
            let lnglats = event.latLngs.map(
                coordinate => [coordinate.lng, coordinate.lat]);
            let drawnPolygon = multiPolygon(
                [[[...lnglats, lnglats[0]]]]);
            // TODO: use turf to simplify polygon
            mapSettings.drawnPolygons.push(drawnPolygon);
            updatePolygon();
        }
    }

    async function updateLocStrings(event) {
        submitDisabled = true;
        await updateStringPolygons(mapSettings, event.detail.tags);
        updatePolygon();
        submitDisabled = false;
    }

    function showPolygonOnMap() {
        previewPolyGroup.clearLayers();
        if (mapSettings.polygon) {
            let map_poly = L.geoJSON(mapSettings.polygon, {
                style: {
                    fillOpacity: 0,
                },
            }).addTo(previewPolyGroup);
            previewMap.fitBounds(map_poly.getBounds());
        }
    }

    async function updatePolygon() {
        submitDisabled = true;
        updateSettingsPolygon(mapSettings);
        showPolygonOnMap();
        submitDisabled = false;
    }

    function exportMapToJson() {
        mapJson = mapSettingsToJson(mapSettings);
    }

    async function importMapFromJson() {
        if (!mapJson) {
            return;
        }
        mapSettings = await mapSettingsFromJson(mapJson);
        updatePolygon();
    }

</script>

<style>
    #locstrings :global(.svelte-tags-input-layout) {
        border-top-left-radius: 0;
        border-top-right-radius: 4px;
        border-bottom-right-radius: 4px;
        border-bottom-left-radius: 0;
        flex: 1 1 0;
        border: 1px solid #ced4da;
    }

    #locstrings :global(.svelte-tags-input-tag) {
        background-color: #007bff;
    }
</style>

<main>
    <div class="container">

    <br>

    <h2>Create a New Map</h2>

    <br>

    <form on:submit|preventDefault={handleFormSubmit} method="post">

        <div class="form-group">
            <div class="input-group">
                <div class="input-group-prepend">
                    <div class="input-group-text">Map Name</div>
                </div>
                <input type="text" class="form-control" id="Name" required bind:value={mapSettings.name}/>
            </div>
        </div>

        <div class="form-group">
            <div class="input-group">
                <div class="input-group-prepend">
                    <div class="input-group-text">Number of Rounds</div>
                </div>
                <input type="number" class="form-control" id="NumRounds" bind:value={mapSettings.numRounds} min="1" max="100"/>
            </div>
        </div>

        <div class="form-row">
            <div class="col">
                <div class="input-group">
                    <div class="input-group-prepend">
                        <div class="input-group-text">Round Time, Minutes</div>
                    </div>
                    <input type="number" min="0" class="form-control mr-sm-3" id="TimeLimit_minutes" bind:value={mapSettings.timeLimitMinutes}/>
                </div>
            </div>
            <div class="col">
                <div class="input-group">
                    <div class="input-group-prepend">
                        <div class="input-group-text">Seconds</div>
                    </div>
                    <input type="number" min="0" class="form-control" id="TimeLimit_seconds" bind:value={mapSettings.timeLimitSeconds}/>
                </div>
            </div>
        </div>
        <small class="form-text text-muted">
            Leave zero for no time limit.
        </small>

        <br/>

        <div class="card border-info">
            <div class="card-header">
                <button class="btn btn-info" type="button" on:click={() => {advancedHidden = !advancedHidden; setTimeout(function() {previewMap.invalidateSize()}, 400)}}>
                    Show advanced settings
                </button>
            </div>

            <div class="card-body" id="advanced-settings" hidden={advancedHidden}>
                <div class="form-group">
                    <div class="input-group">
                        <div class="input-group-prepend">
                            <div class="input-group-text">Grace Distance (m)</div>
                        </div>
                        <input type="number" class="form-control" id="GraceDistance" bind:value={mapSettings.graceDistance} min="0"/>
                    </div>
                </div>
                <small class="form-text text-muted">
                    Guesses within this distance (in meters) will be awarded full points.
                </small>
                <hr/>
                <!-- TODO: it would be nice if this was a double range slider -->
                <div class="form-row">
                    <div class="col">
                        <div class="input-group">
                            <div class="input-group-prepend">
                                <div class="input-group-text">Population Density %, Minimum</div>
                            </div>
                            <input type="number" class="form-control mr-sm-3" id="MinDensity" bind:value={mapSettings.minDensity} min="0" max="100"/>
                        </div>
                    </div>
                    <div class="col">
                        <div class="input-group">
                            <div class="input-group-prepend">
                                <div class="input-group-text">Maximum</div>
                            </div>
                            <input type="number" class="form-control mr-sm-3" id="MaxDensity" bind:value={mapSettings.maxDensity} min="0" max="100"/>
                        </div>
                    </div>
                </div>
                <small class="form-text text-muted">
                    0% is ocean. 10% is barren road. With 20%, you will find signs of civilization. Anything above 50% is already very populated.
                </small>

                <hr/>

                <div class="form-group">
                    <div class="input-group">
                        <div class="input-group-prepend">
                            <div class="input-group-text">Panorama connectedness</div>
                        </div>
                        <!-- note: select values are Object.  Wrapping them in
                                   brackets takes advantage of object init
                                   shorthand to give us ints instead of strings.
                                   However! The resulting binding is not 
                                   bidirectional, so make sure your mapSettings
                                   defaults match the select defaults. -->
                        <select class="form-control" id="Connectedness" bind:value={mapSettings.connectedness}>
                            <option value={1} selected="selected">always</option>
                            <option value={2} >never</option>
                            <option value={0} >any</option>
                        </select>
                    </div>
                </div>
                <small class="form-text text-muted">
                    If you want to be able to always walk somewhere or if you want single-image ones. 
                </small>

                <hr/>

                <div class="form-group">
                    <div class="input-group">
                        <div class="input-group-prepend">
                            <div class="input-group-text">Copyright</div>
                        </div>
                        <select class="form-control" id="Copyright" bind:value={mapSettings.copyright}>
                            <option value={0} selected="selected">any</option>
                            <option value={1}>Google only</option>
                            <option value={2}>third party only</option>
                        </select>
                    </div>
                </div>
                <small class="form-text text-muted">
                    If you want to see only Google panos or also include third party panos.
                </small>

                <hr/>

                <div class="form-group">
                    <div class="input-group">
                        <div class="input-group-prepend">
                            <div class="input-group-text">Source</div>
                        </div>
                        <select class="form-control" id="Source" bind:value={mapSettings.source}>
                            <option value={1} selected="selected">outdoors only</option>
                            <option value={0}>any</option>
                        </select>
                    </div>
                </div>
                <small class="form-text text-muted">
                    If you want to exclude panoramas inside businesses.
                </small>

                <hr/>

                <div class="form-group">
                    <div class="form-check form-check-inline">
                        <input class="form-check-input" type="checkbox" id="ShowLabels" bind:checked={mapSettings.showLabels}>
                        <label class="form-check-label" for="label">Show labels on map</label>
                    </div>
                </div>
                <small class="form-text text-muted">
                    Check this if the map should tell you how places are called.
                </small>

                <hr/>
                
                <div class="form-group">
                    <div class="input-group" id="locstrings">
                        <div class="input-group-prepend">
                            <div class="input-group-text">Location string </div>
                        </div>
                        <Tags on:tags={updateLocStrings}/>
                    </div>
                    <small class="form-text text-muted">
                        Constrain the game to the specified places - countries, states, cities, neighborhoods, or any other bounded areas.  
                        You can add multiple locations - press Enter to add each string.
                    </small>
                    <div class="card bg-danger text-white mt-1" id="error-dialog" hidden>
                        <p class="card-text">Sorry, that does not seem like a valid bounding box on OSM Nominatim.</p>
                    </div>
                    <div class="btn-toolbar mt-3">
                        <div class="btn-group mr-2">
                            <button class="btn btn-outline-info" type="button" on:click={() => {lasso.enable()}}>
                                Add area by drawing
                            </button>
                            <button class="btn btn-outline-danger" type="button" on:click={() => {mapSettings.drawnPolygons.pop(); updatePolygon()}}>
                                Undo
                            </button>
                        </div>
                        <div class="btn-group">
                            <button class="btn btn-outline-danger" type="button" on:click={() => {mapSettings.drawnPolygons = []; updatePolygon()}}>
                                Clear all drawn areas
                            </button>
                        </div>
                    </div>
                </div>
                <div id="bounds-map" style="width: 80%; height: 50vh; margin-left: 10%; margin-right: 10%;"></div>
            
                <hr/>

                <div class="form-group">
                    <div class="input-group">
                        <div class="btn-group mr-2">
                            <button class="btn btn-info" type="button" on:click={exportMapToJson}>Export Map to JSON</button>
                            <button class="btn btn-danger" type="button" on:click={importMapFromJson}>Import Map from JSON</button>
                        </div>
                    </div>
                    <textarea class="form-control" rows=10 bind:value={mapJson}></textarea>
                </div>
            </div>
        </div>

        <br/>

        <input id="hidden-input" type="hidden" name="result" value=""/>

        <button id="submit-button" type="submit" class="btn btn-primary" style="margin-bottom: 2em;" disabled={submitDisabled}>Create Map</button>

    </form>
    <!-- <link rel="stylesheet" href="public/leaflet/leaflet.css"/> -->
    </div>
</main>
