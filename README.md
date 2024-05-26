# Trollinfo

Sending you information about your trolls. Written in go in a single day, no guarantees anything is working here.

Sends a message with arriving, staying and leaving trolls to a matrix channel 14 minutes before a shift starts. Further, displays a web view with the latest state of arriving, staying, leaving trolls at `/?token={your-token}` (use `refresh_seconds=2` parameter to auto-refresh the page) and exposes the data as JSON on `/data?token={your-token}`.

## Configure

Set the following environment variables:

| Name                          | Description                                                                             |
| ----------------------------- | --------------------------------------------------------------------------------------- |
| `TROLLINFO_API_BASE_URL`      | Base URL (domain) where your Engelsystem runs at                                        |
| `TROLLINFO_API_KEY`           | API key for accessing the Engelsystem API                                               |
| `TROLLINFO_LOCATIONS`         | Comma separated locations used to query shifts for                                      |
| `TROLLINFO_MATRIX_ROOM_ID`    | Matrix room ID to send messages to                                                      |
| `TROLLINFO_HTTP_LISTEN_ADDR`  | HTTP server listen address to expose data to                                            |
| `TROLLINFO_HTTP_TOKEN`        | Token to secure HTTP served data (use as query paramter named `token`)                  |
| `TROLLINFO_MATRIX_USERNAME`   | Matrix username of the account to send messages with, excluding the home server address |
| `TROLLINFO_MATRIX_PASSWORD`   | Matrix user password                                                                    |
| `TROLLINFO_MATRIX_HOMESERVER` | Matrix user homeserver                                                                  |
| `TROLLINFO_MATRIX_DEVICE_ID`  | Unique device ID used for connection to matrix server                                   |
