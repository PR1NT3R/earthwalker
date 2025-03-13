<script>
    import { get } from 'svelte/store';
    import { loc, globalChallenge, globalResult, ewapi } from '../js/stores.js';
    import MapPreview from './components/MapPreview.svelte';

    let allMaps = [];

    async function fetchData() {
        let maps = await $ewapi.getMaps()
        allMaps =  maps.slice(0, 20);
    }

    var ip = null

    async function getIp() {
        if (ip !== null) return ip; // Use cached value if available

        try {
            const response = await fetch("/api/my-ip");
            const data = await response.json();
            ip = data.ip || ""; // Ensure ip is always set to a string
            // console.log("User IP:", ip); // Debugging log
            return ip;
        } catch (error) {
            // console.error("Failed to fetch IP:", error);
            return ""; // Return empty string if fetching fails
        }
    }

    async function remoteMapCreationAllowed() {
        let allowedStr = (await $ewapi.getRemoteMapCreationAllowed()).allowremotemapcreation;
        // console.log(JSON.parse(allowedStr.toLowerCase()));
        return JSON.parse(allowedStr.toLowerCase())
    }

    async function isIpAllowed() {
        const userIp = await getIp();

        // Fetch allowed IPs from the backend
        let allowedIps = [];
        try {
            const response = await fetch("/api/allowed-ips");
            const data = await response.json();
            allowedIps = data || [];
        } catch (error) {
            console.error("Error fetching allowed IPs:", error);
        }

        return allowedIps.includes(userIp);
    }
</script>

<main>
    {#if $globalChallenge && $globalResult}
        <a href={"/play?id=" + $globalChallenge.ChallengeID} class="btn btn-primary">Resume Game</a>
        <p>Challenge ID: <code>{$globalChallenge.ChallengeID}</code>, Result ID: <code>{$globalResult.ChallengeResultID}</code></p>
        <hr/>
    {:else}
        <p>No game in progress.</p>
    {/if}
    <h2>Maps</h2>
    {#await isIpAllowed() then ipAllowed}
        {#await remoteMapCreationAllowed() then remoteCreationAllowed}
            {#if ipAllowed || remoteCreationAllowed}
                <p on:click={() => {$loc = "/createmap";}} class="btn btn-primary">New Map</p>
            {/if}
        {/await}
    {/await}
    {#await fetchData()}
        <h2>Loading...</h2>
    {:then}
        <div class="container-fluid mb-4">
            <div class="row justify-content-center row-cols-sm-2 row-cols-md-3 row-cols-lg-4 row-cols-xl-5">
                {#each allMaps as map}
                    <MapPreview on:requestReload={fetchData} {map}/>
                {/each}
            </div>
        </div>
    {/await}
</main>