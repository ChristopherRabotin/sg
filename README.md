# Stress Gauge - sg
sg allows one to gauge response times of an HTTP service under stress.

[![Build Status](https://travis-ci.org/ChristopherRabotin/sg.svg?branch=master)](https://travis-ci.org/ChristopherRabotin/sg) [![Coverage Status](https://coveralls.io/repos/ChristopherRabotin/sg/badge.svg?branch=master&service=github)](https://coveralls.io/github/ChristopherRabotin/sg?branch=master)
[![goreport](https://goreportcard.com/badge/github.com/ChristopherRabotin/sg)](https://goreportcard.com/report/github.com/ChristopherRabotin/sg)

# Features
*Note:* what is in italics is not yet implemented.
 - XML test profile;
 - XML result file, with XSL for humans to read;
 - Set total number of requests and total number of concurrent requests;
 - Response time break down by percentile;
 - Set header, body and cookie(s) from an initial request or within XML;
 - Regex-like URL generation.

# Quick start
Grab the [basic example](docs/examples/basic.xml) and start changing with the test profile.
