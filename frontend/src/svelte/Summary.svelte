<script>
    // TODO: most of this script is duplicated in Scores.svelte.
    //       (also a bit in Modify.svelte)
    //       consolidate.
    import {onMount} from 'svelte';
    import { ewapi, globalMap, globalChallenge, globalResult } from '../js/stores.js';
    import LeafletGuessesMap from './components/LeafletGuessesMap.svelte';
    import Leaderboard from './components/Leaderboard.svelte';
    import utils from '../js/utils';
    import { calcScoreDistance, distString } from '../js/earthwalker';

    let displayedResults;
    let allResults = [];

    let guessLocs;
    let actualLocs;
    let scoreDists = [];

    let gameLink = utils.getGameLink($globalChallenge.ChallengeID);

    // leaflet
    let scoreMap;
    let scoreMapPolyGroup;
    let scoreMapGuessGroup;

    async function fetchData() {
        allResults = await $ewapi.getAllResults($globalChallenge.ChallengeID);
        allResults.forEach(r => {
            r.scoreDists = r.Guesses.map((guess, i) => calcScoreDistance(guess, $globalChallenge.Places[i], $globalMap.GraceDistance, $globalMap.Area));
            r.scoreDists = r.scoreDists.concat(Array($globalMap.NumRounds - r.scoreDists.length).fill([0, 0]));
            r.totalScore = r.scoreDists.reduce((acc, val) => acc + val[0], 0);
            r.totalDist = r.scoreDists.reduce((acc, val) => acc + val[1], 0)
        });
        allResults.sort((a, b) => b.totalScore - a.totalScore);
        allResults = allResults;
        displayedResults = allResults;
    }
</script>

<!-- This prevents users who haven't finished the challenge from viewing
     TODO: cleaner protection for this page -->
{#if $globalResult.Guesses && $globalMap.NumRounds && $globalResult.Guesses.length == $globalMap.NumRounds}
    {#await fetchData()}
        <h2>Loading...</h2>
    {:then}
        <LeafletGuessesMap {displayedResults} showAll={true}/>

        <div class="container">
            <br>
            <div class="row justify-content-center">
                <div class="input-group w-50">
                    <input type="text" class="form-control" readonly="readonly" bind:value={gameLink} disabled={!gameLink} />
                    <div class="input-group-append">
                        <button type="button" id="copy-game-link" class="btn btn-primary" on:click={() => utils.copyToClipboard(gameLink)} disabled={!gameLink}>
                            &#128203;
                        </button>
                    </div>
                </div>
            </div>

            <div id="leaderboard" style="margin-top: 2em; text-align: center;">
                <h3>Challenge Leaderboard</h3>
                <Leaderboard bind:displayedResults={displayedResults} {allResults} curRound={$globalMap.NumRounds - 1}/>
            </div>

            <div style="margin-top: 2em; text-align: center;">
                {#each displayedResults as displayedResult, j}
                <h3>{displayedResult && displayedResult.Nickname ? displayedResult.Nickname + "\'s" : "Your"} scores:</h3>
                <table class="table table-striped">
                    <thead>
                    <th scope="col">Round</th>
                    <th scope="col">Points</th>
                    <th scope="col">Distance Off</th>
                    </thead>
                    <tbody>
                    {#if displayedResult && displayedResult.scoreDists}
                        {#each displayedResult.scoreDists as scoreDist, i}
                            <tr scope="row">
                                <td>{i + 1}</td>
                                <td>{scoreDist[0]}</td>
                                <td>{distString(scoreDist[1])}</td>
                            </tr>
                        {/each}
                    {/if}
                    </tbody>
                </table>
                {/each}
            </div>
        </div>
    {/await}
{:else}
    <div class="text-center">
        <h2>You must finish the game to view this page.</h2>
    </div>
{/if}
