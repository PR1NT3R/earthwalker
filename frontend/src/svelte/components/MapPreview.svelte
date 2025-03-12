<script>
    import { onMount, createEventDispatcher } from "svelte";
    import { ewapi } from "../../js/stores";
    import { showPolygonOnMap } from '../../js/earthwalker';

    const dispatch = createEventDispatcher();

    export let map;

    let tileServer;
    let mapDiv;

    let lMap;
    let polyGroup;

    onMount(async () => {
        lMap = new L.Map(mapDiv, {
            attributionControl: false,
            zoomControl: false
        });
        lMap.setView([0, 0], 0);

        tileServer = (await $ewapi.getTileServer()).tileserver;
        L.tileLayer(tileServer).addTo(lMap);

        polyGroup = L.featureGroup().addTo(lMap);
        if (map.Polygon) {
            showPolygonOnMap(polyGroup, map.Polygon);
            lMap.fitBounds(polyGroup.getBounds());
        }
    });

    const LOCAL_HOSTS = ["localhost", "127.0.0.1", "192.168.0.127"] // not exhaustive
    function isLocalHost() {
        console.log(LOCAL_HOSTS.includes(location.hostname));
        return LOCAL_HOSTS.includes(location.hostname)
    }

    async function remoteMapDeletionAllowed() {
        let allowedStr = (await $ewapi.getRemoteMapDeletionAllowed()).allowremotemapdeletion;
        console.log(JSON.parse(allowedStr.toLowerCase()));
        return JSON.parse(allowedStr.toLowerCase())
    }

    async function remoteMapCreationAllowed() {
        let allowedStr = (await $ewapi.getRemoteMapCreationAllowed()).allowremotemapcreation;
        console.log(JSON.parse(allowedStr.toLowerCase()));
        return JSON.parse(allowedStr.toLowerCase())
    }

    async function deleteSelf() {
        if (confirm("Are you sure? This action is PERNAMENT.")) {
            let response = await $ewapi.deleteMap(map.MapID);
            console.log(response);
            dispatch("requestReload");
        }
    }
</script>

<div class="card mt-4 mx-1">
    <div 
        bind:this={mapDiv}
        id="map"
        style="min-height: 5rem; width: 100%;"
    ></div>
    <div class="card-body">
        <h5 class="card-title">{map.Name}</h5>
        <p class="card-text">
            Rounds: {map.NumRounds}
            <br>
            {map.TimeLimit > 0 ? `Time limit: ${Math.floor(map.TimeLimit / 60)}:${Math.floor(map.TimeLimit % 60).toString().padStart(2, '0')}` : 'No Time Limit'}
        </p>
    </div>
    <div class="card-footer">
        <a href="/createchallenge?mapid={map.MapID}" class="btn btn-primary">
            Use Map
        </a>
        {#await remoteMapDeletionAllowed() then remoteDeletionAllowed}
            {#if isLocalHost() || remoteDeletionAllowed}
                <button class="btn btn-danger" on:click|preventDefault={deleteSelf}>
                    Delete
                </button>
            {/if}
        {/await}
    </div>
</div>