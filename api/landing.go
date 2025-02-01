package handler

import (
	"fmt"
	"net/http"
	"text/template"
)

// LandingHandler serves the landing page
func LandingHandler(w http.ResponseWriter, r *http.Request) {
	// Define the HTML template
	tmpl := `
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Welcome to Siri Playlist Actions</title>
			<style>
				body {
					font-family: Arial, sans-serif;
					margin: 0;
					padding: 0;
					display: flex;
					justify-content: center;
					align-items: center;
					height: 100vh;
					background-color: #f4f4f4;
				}
				.container {
					max-width: 600px;
					background: white;
					padding: 20px;
					border-radius: 10px;
					box-shadow: 0px 4px 8px rgba(0, 0, 0, 0.2);
					text-align: center;
				}
				h1 {
					color: #333;
				}
				p {
					font-size: 18px;
					color: #666;
					margin-bottom: 20px;
				}
				.get-started {
					background-color: #007BFF;
					color: white;
					border: none;
					padding: 12px 20px;
					font-size: 18px;
					border-radius: 5px;
					cursor: pointer;
					text-decoration: none;
					display: inline-block;
				}
				.get-started:hover {
					background-color: #0056b3;
				}
			</style>
		</head>
		<body>
			<div class="container">
				<h1>Welcome to Siri Playlist Actions 🎶</h1>
				<p>Seamlessly integrate your Spotify experience with Siri Shortcuts!</p>
				<p>This free community-supported service allows you to add songs to playlists, and remove them—all with voice commands.</p>
				<p>To get started, connect to your Spotify account:</p>
				<a class="get-started" href="/api/login">Connect to Spotify</a>
			</div>
		</body>
		</html>
	`

	// Parse and execute the template
	t, err := template.New("landing").Parse(tmpl)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing template: %s", err), http.StatusInternalServerError)
		return
	}

	// Render the template
	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error rendering template: %s", err), http.StatusInternalServerError)
	}
}
