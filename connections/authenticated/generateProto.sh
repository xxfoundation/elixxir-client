#!/bin/bash

protoc --go_out=paths=source_relative:. connections/authenticated/authenticated.proto
