<script>
    import { onMount } from 'svelte';
    import { ewapi, globalMap, globalChallenge, globalResult } from '../js/stores.js';
    import MapInfo from './components/MapInfo.svelte';

    const challengeCookieName = "earthwalker_lastChallenge";
    const resultCookiePrefix = "earthwalker_lastResult_";

    let nickname = "";

    // TODO: this duplicates a function in CreateChallenge.
    //       consider consolidating.
    async function handleFormSubmit() {
        nickname = nickname.substring(0,20);
        $globalResult = await $ewapi.getResult(await submitNewChallengeResult());
        // set the generated challenge as the current challenge
        document.cookie = challengeCookieName + "=" + $globalChallenge.ChallengeID + ";path=/;max-age=172800";
        // set the generated ChallengeResult as the current ChallengeResult
        // for the Challenge with challengeID
        document.cookie = resultCookiePrefix + $globalChallenge.ChallengeID + "=" + $globalResult.ChallengeResultID + ";path=/;max-age=172800";
        window.location.replace("/play");
    }

    // TODO: this duplicates a function in CreateChallenge.
    //       consolidate to api lib
    async function submitNewChallengeResult() {
        let challengeResult = {
            ChallengeID: $globalChallenge.ChallengeID,
            Nickname: nickname,
        };
        let data = await $ewapi.postResult(challengeResult);
        return data.ChallengeResultID;
    }

</script>

<main>
    {#if $globalChallenge && $globalChallenge.ChallengeID}
        <form on:submit|preventDefault={handleFormSubmit} class="container">
            <br>
            <h2>Join Challenge</h2>
            <p>Playing: {$globalMap.Name}<MapInfo/></p>
            <div action="">
                <!-- TODO: show map settings -->
                <div class="form-group">
                    <div class="input-group">
                        <div class="input-group-prepend">
                            <div class="input-group-text">Your Nickname</div>
                        </div>
                        <input bind:value={nickname} required type="text" class="form-control" id="Nickname"/>
                    </div>
                </div>

                <button id="submit-button" class="btn btn-primary" style="margin-bottom: 2em; color: #fff;">Start Challenge</button>

            </div>
        </form>
    {:else}
        <h2>No Challenge Found</h2>
        <p>Please make sure the URL contains a valid Challenge ID.</p>
    {/if}
</main>