# siri-playlist-actions
Custom siri actions to interact with the current Spotify song or playlist


## Local Development

Create a .env file

    cp .env.example .env

Update values in .env with your spotify developer credentials and redis url found on the Vercel dashboard page.


Run the server:

    export $(cat .env | xargs) && vercel dev --listen 8080
