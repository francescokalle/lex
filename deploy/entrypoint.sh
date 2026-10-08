#!/bin/sh
set -e

DATA_DIR="${DATA_DIR:-/data}"
SITE_PORT="${SITE_PORT:-8080}"

# Download dataset if not present
if [ ! -d "$DATA_DIR/graph" ]; then
    echo "lex: downloading dataset..."
    /app/lex -data "$DATA_DIR" 2>&1 | head -5
fi

# Start a simple static file server for the site in background
if command -v python3 >/dev/null 2>&1; then
    cd /app/site
    python3 -m http.server "$SITE_PORT" &
    SERVER_PID=$!
    echo "lex: site serving on http://localhost:$SITE_PID"
    wait $SERVER_PID
else
    # Just run the lex server over stdio if no site server available
    cd /app
    exec /app/lex -data "$DATA_DIR"
fi
