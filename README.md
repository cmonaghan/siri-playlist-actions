# siri-playlist-actions

Custom siri actions to interact with the current Spotify song or playlist


## Local Development

Create a .env.local file

    vercel env pull

If you want to test login to spotify while running locally, you'll also need to:

1. Update your .env.local to `REDIRECT_URI=http://localhost:8080/api/callback`
2. In the [spotify developer portal](https://developer.spotify.com/dashboard/645f0d6f7ba34906b685002e1308be1c/settings), update the redirect uri to `http://localhost:8080/api/callback`

Though if you already have a valid API key these steps are not necessary.


Run the server:

    vercel dev --listen 8080


## Deploy

    vercel deploy
