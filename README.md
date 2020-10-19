# MB8600 telemetry to influxdb

This project marries the clever work done in
https://github.com/AdamJacobMuller/mb8600, which allows us to gather
statistics without a password using go to the schema which was put
together here in https://github.com/mattund/modem-statistics, which creates a beatiful dashboard like so:

![Screenshot of dashboard](https://camo.githubusercontent.com/f24a3eaafd1f4ac397f5b888b7c62c7efb366901/68747470733a2f2f692e696d6775722e636f6d2f3049764471656a2e706e67)

You probably want to use https://github.com/artbird309/MB8600-Docker-Image, to build a docker container to run this script in.  You'll also need have a influxdb 1.x server and the grafana server running to get everything to work.