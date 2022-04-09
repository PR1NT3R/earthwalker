<script>
    import { loc, globalChallenge, globalResult, ewapi } from '../js/stores.js';
    import MapPreview from './components/MapPreview.svelte';

    let allMaps = [];

    async function fetchData() {
        let maps = await $ewapi.getMaps()
        allMaps =  maps.slice(0, 20);
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
    <p on:click={() => {$loc = "/createmap";}} class="btn btn-primary">New Map</p>
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