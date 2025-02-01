# siri-playlist-actions

Custom siri actions to interact with the current Spotify song or playlist


## Local Development

Create a .env.local file from existing vercel environment variables

    vercel env pull

The best way to set a variable differently for dev vs prod is to set it in Vercel, and then specify the particular environment for variable. For example:

| Environment | Key          | Value                                            |
|-------------|--------------|--------------------------------------------------|
| Development | REDIRECT_URI | http://localhost:8080/api/callback             |
| Production  | REDIRECT_URI | https://spotify.woolgathering.io/api/callback  |


Run the server:

    vercel dev --listen 8080

Note that `vercel dev` actually pulls the environment variables from vercel, and does not respect your .env.local file (annoying).

Navigate to http://localhost:8080/ and click "Connect to Spotify", which will redirect you to the setup page with instructions. 

The best way to test changes is to setup Apple Shortcuts as described on the setup page. However, you can also use `curl`, like so:

Endpoint: `/api/current-song`

    curl -X GET "http://localhost:8080/api/current-song" \
     -H "X-API-Key: YOUR_API_KEY"

Endpoint: `/api/add-song`

    curl -X POST "http://localhost:8080/api/add-song" \
     -H "Content-Type: application/json" \
     -H "X-API-Key: YOUR_API_KEY" \
     -d '{"playlist_id": "YOUR_PLAYLIST_ID"}'

Endpoint: `/api/remove-song`

    curl -X DELETE "http://localhost:8080/api/remove-song" \
     -H "X-API-Key: YOUR_API_KEY"

Endpoint: `/api/revoke`

    curl -X POST http://localhost:8080/api/revoke \
     -H "X-API-Key: YOUR_API_KEY"


## Deploy

To prod:

    vercel deploy --prod

Note that a redeployment is necessary after changing environment variables.
