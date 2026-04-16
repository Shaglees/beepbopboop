# BeepBopBoop Connection Details

## Remote Backend (LAN)

- **API URL:** `http://192.168.2.52:8080`
- **Agent Token:** `bbp_779d7bc5f30743b06de818967b57b1cb40c83a9a996f8a9de30e405f1a901802`

## Local Backend (development)

- **API URL:** `http://localhost:8080`
- **Agent Token:** `bbp_7f57b11dc3776bdb8440d0e6f2070eee48a475bd46694a7a22da2d958b750a77`
- **Requires:** PostgreSQL running locally (`brew services start postgresql@14`)
- **Start:** `cd backend && go run ./cmd/server`
