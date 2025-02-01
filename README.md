# siri-playlist-actions

Custom siri actions to interact with the current Spotify song or playlist


## Local Development

Create a .env.local file

    vercel env pull

The best way to set a variable differently for dev vs prod is to set it in Vercel, and then specify the particular environment for variable. For example:

| Environment | Key          | Value                                            |
|-------------|--------------|--------------------------------------------------|
| Development | REDIRECT_URI | "http://localhost:8080/api/callback"             |
| Production  | REDIRECT_URI | "https://spotify.woolgathering.io/api/callback"  |


Run the server:

    vercel dev --listen 8080


## Deploy

To prod:

    vercel deploy --prod
