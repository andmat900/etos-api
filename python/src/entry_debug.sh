#!/bin/bash

exec uvicorn etos_api.main:APP \
	--host 0.0.0.0 \
	--port 8004 \
	--reload
