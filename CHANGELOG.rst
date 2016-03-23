########
CHANGELOG
########

UPDATE
	backword compatible change
ADD
	backword compatible enhancement
CHANGE
	backworkd incompatible change
FIX
	bug fix

1.0.2
=====

:release: 2016-03-23

UPDATE

- build: switched from godep to glide.

FIX

- device: serial subscribe topic mismatch

1.0.1
=====

:release: 2016-02-29

UPDATE

- build: Golang version 1.6

1.0.0
=====

:release: 2016-02-15

UPDATE

- build: Golang version 1.5.3
- build: Remove gopsutil/common from Godep.json.

ADD

- config: validate and abort in case of invalid configuration.
- status: status report on IP address.
- gateway: MQTT client authentication using clients certification.
- gateway: added test in the case of empty gateway name.
- gateway: will_topic configuration capability.
- test: TLS testcase added using mosquitto broker on docker.

CHANGE

- config: configuration syntax modified to use type string to show device type.
- config: fuji exits when there are irregular configuration formats in toml.
- config: switch configuration file format from ini to toml.
- gateway: publish/subscribe topic changed to distinguish both.

FIX

- test: will testcase fix.

0.3.0
=====

:release: 2015-10-07

UPDATE

- build: refactored Makefile
- build: change to go 1.5.1
- build: generated control from control.in.
- build: update library dependency on paho and gopsutil.
- optimization: call regexp.MustCompile() once on start-up.
- refactored: moved inidef/porttype.go to device/porttype.go.

ADD

- app: customized cli.VersionPrinter.

FIX

- fix DeviceChannel multiplexing


0.2.3
=====

:release: 2015-05-19

UPDATE

- document: add "How To Release" to README.rst.

FIX

- build: fixed version TAG and CHANGELOG description.


0.2.2
======

:release: 2015-05-19

FIX

- device: remove default clause to not consume all cpu.

0.2.1
=====

:release: 2015-04-29

FIX

- build: fixed ARM5 and ARM7 build settings.

0.2.0
======

First public release

:release: 2015-04-22
