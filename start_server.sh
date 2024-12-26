#!/bin/bash

# Start the uvicorn server and run it in the background
poetry run uvicorn --app-dir examples/python_code_agent main:app --port 9200 & 

# Wait a few seconds to ensure the first server is up
sleep 3

# Start the Go server
go run .

# The script will keep running until you press Ctrl+C
# When you do, this will kill both processes
trap "kill 0" EXIT
