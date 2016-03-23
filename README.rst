###########################
MQTT Gateway: Fuji
###########################

:version: 1.0.2

.. image:: https://circleci.com/gh/shiguredo/fuji/tree/develop.svg?style=svg&circle-token=203d959fffaf8dcdc0c68642dde5329e55a47792
    :target: https://circleci.com/gh/shiguredo/fuji/tree/develop

What is MQTT Gateway
=====================

**This definition is Shiguredo original**

A MQTT gateway is a sensor-MQTT gateway which receives data from sensors and sends that data to a MQTT broker.

Architecture::

    <sensor> -> (BLE)     ->             +-------+
    <sensor> -> (EnOcean) -> (Serial) -> |Gateway| -> (MQTT) -> <MQTT Broker>
    <sensor> -> (USB)     ->             +-------+

fuji is a MQTT gateway which is written by Golang.

Supported Hardware
====================

- `Raspberry Pi <http://www.raspberrypi.org/>`_ series
- `Armadillo-IoT <http://armadillo.atmark-techno.com/armadillo-iot>`_
- `Intel Edison <http://www.intel.com/content/www/us/en/do-it-yourself/edison.html?_ga=1.251267654.1109522025.1429502791>`_
- Linux amd64/i386

Coming Soon

- Mac OS X
- FreeBSD
- Windows (7 or later)

Downloads
=========

:URL: https://github.com/shiguredo/fuji/releases/tag/1.0.2

- `fuji-gw_1.0.2_arm5.tar.gz <https://github.com/shiguredo/fuji/releases/download/1.0.2/fuji-gw_1.0.2_arm5.tar.gz>`_
- `fuji-gw_1.0.2_arm6.tar.gz <https://github.com/shiguredo/fuji/releases/download/1.0.2/fuji-gw_1.0.2_arm6.tar.gz>`_
- `fuji-gw_1.0.2_arm7.tar.gz <https://github.com/shiguredo/fuji/releases/download/1.0.2/fuji-gw_1.0.2_arm7.tar.gz>`_
- `fuji-gw_1.0.2_edison_386.ipk <https://github.com/shiguredo/fuji/releases/download/1.0.2/fuji-gw_1.0.2_edison_386.ipk>`_
- `fuji-gw_1.0.2_linux_386.tar.gz <https://github.com/shiguredo/fuji/releases/download/1.0.2/fuji-gw_1.0.2_linux_386.tar.gz>`_
- `fuji-gw_1.0.2_linux_amd64.tar.gz <https://github.com/shiguredo/fuji/releases/download/1.0.2/fuji-gw_1.0.2_linux_amd64.tar.gz>`_
- `fuji-gw_1.0.2_raspi2_arm7.deb <https://github.com/shiguredo/fuji/releases/download/1.0.2/fuji-gw_1.0.2_raspi2_arm7.deb>`_
- `fuji-gw_1.0.2_raspi_arm6.deb <https://github.com/shiguredo/fuji/releases/download/1.0.2/fuji-gw_1.0.2_raspi_arm6.deb>`_

ChangeLog
=========

see `CHANGELOG.rst <https://github.com/shiguredo/fuji/blob/develop/CHANGELOG.rst>`_

Migration Notice
================

from version 0.3.0 or under to 1.0.2
--------------------------------------

Configuration file format changed to TOML

see  `config.toml.example <https://github.com/shiguredo/fuji/blob/develop/config.toml.example>`_


Topic of publish/subscribe message changed

- publish topic format from device ::

     <topicprefix>/<gatewayname>/<devicename>/<devicetype>/publish

- subscribe topic format to device (currently serial type device only support this functionality) ::

     <topicprefix>/<gatewayname>/<devicename>/<devicetype>/subscribe


Build
=====

see `BUILD.rst <https://github.com/shiguredo/fuji/blob/develop/BUILD.rst>`_

Install
=======

see `INSTALL.rst <https://github.com/shiguredo/fuji/blob/develop/INSTALL.rst>`_

How to Contribute
=================

see `CONTRIBUTING.rst <https://github.com/shiguredo/fuji/blob/develop/CONTRIBUTING.rst>`_

Development Logs
========================

**Sorry for written in Japanese.**

**開発に関する詳細については開発ログをご覧ください**

`時雨堂 MQTT ゲートウェイ Fuji 開発ログ <https://gist.github.com/voluntas/23132cd3848af5b3ee1e>`_


How To Release
==================

1. git flow release start x.y.z
2. update TAG in Makefile
3. update CHANGELOG.rst and README.rst
4. git commit
5. git flow release finish x.y.z
6. git push

License
========

::

  Copyright 2015-2016 Shiguredo Inc. <fuji@shiguredo.jp>

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
