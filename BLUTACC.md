# The BLUTACC (Bluetooth Terra AC Control) Vulnerability

Discovered by duck. (@duckfullstop) and puck (@puckipedia)

## Pretext:

The ABB Terra AC line of Electric Vehicle Charging Points is a popular series of vehicle charging units used around the world, and is used by a number of service providers as well as by domestic end users.

The lineup features numerous connectivity options for configuration and management. These include:

- On all models:
  - Wifi
  - Bluetooth
  - Ethernet
  - Modbus
- On selected models:
  - RFID (for charge session authentication)
  - 4G Cellular via SIM card slot
  - Secondary Ethernet port for daisy-chaining

This vulnerability focuses on the Bluetooth interface available on the entire lineup of Terra AC units.

The Bluetooth interface is intended for initial device configuration by an installer or administrator (using the TerraConfig mobile application, registration restricted to registered electricians), as well as for management by a domestic end-user (using the ChargerSync mobile application, open registration). The ChargerSync application has less access than the TerraConfig application, and cannot access options such as OCPP server configuration, current limits, and load balancing.

## Overview:

The BLUTACC vulnerability is an authentication bypass present on ABB's Terra AC line of Electric Vehicle Charging Points.

Using this vulnerability, an attacker may connect to any charge point and access the full range of configuration and setup options afforded to the TerraConfig application, including:

- Device reset
- Arbitrary firmware binary upload
- Enrolment (and disenrolment) of arbitrary RFID cards
- Charge session authorisation without external verification
- Network connectivity settings, including cellular APN
- Open Charge Point Protocl (OCPP) server settings
- Charger current limit configuration modification (potentially allowing unsafe electrical situations)

## Susceptible Devices:

The vulnerability has been tested on the following models:

- Terra AC W7-T-R-0
- Terra AC W7-T-RD-MC-0

as well as the following versions:
- v1.4.2
- v1.5.2
- v1.6.5

We believe that this vulnerability is also present on all models of Terra AC Wallbox, namely (not including the above two models):

- W4-S-0
- W4-S-R-0
- W7-T-0
- W7-T-R-C-0
- W7-G5-R-0
- W7-G5-RD-MC-0
- W11-G5-R-0
- W22-T-0
- W22-T-R-0
- W22-T-R-C-0
- W22-S-R-0
- W22-S-R-C-0
- W22-G5-R-C-0
- W22-T-RD-M-0
- W22-T-RD-MC-0
- W22-S-RD-MC-0
- W22-G5-RD-MC-0
- W7-P8-R-D-0
- W7-P8-R-CD-0
- W7-P8-RD-MD-0
- W7-P9-RD-MCD-0

## The Vulnerability in Detail:

To ensure that only authorised users may connect to a Terra AC charger, the TerraConfig and ChargerSync applications request a PIN - an 8 digit alphanumeric code included in the literature with each unit.

Users sign in to the TerraConfig and ChargerSync applications with a username / password combination assigned to an ABB-side account. These accounts are different for TerraConfig and ChargerSync, and TerraConfig accounts are only provided by ABB to qualified electricians.

When the TerraConfig Application connects to a Terra AC charger, it performs the following steps:

- User selects "Connect" in Application
- Application displays a list of chargers that can be connected to
- Upon selecting a charger, Application prompts for PIN
- Application verifies PIN with a request to ABB server
- If PIN is correct, Application opens a Bluetooth Low Energy session with the charger
- Application authenticates itself with the charger using an encrypted Authorisation command
- If authentication is successful, the charger responds with (alongside other information such as firmware versions) a token to be included with all future commands, and audibly beeps twice
- The Application then proceeds with further commands to determine charger state, perform configuration tasks, etcetera

Commands are sent in a standard binary format, and have a command identifier, authorisation token (set to nil / ignored for the Authorisation command), and a data section.


The BLUTACC vulnerability exploits a major oversight in the authentication procedure, namely with the Authorisation command.

The Authorisation command is the only command which sends data in an encrypted fashion, using the 3DES ECB algorithm (it should be noted that 3DES is a known insecure algorithm - CVE-2016-2183). It does this using a static secret which is stored in the application, and so is easily retrieved via decompilation.

This encrypted data consists of the following:

- The Serial Number of the unit, capitalised without dashes
- The ABB User ID of the application user

The glaring omission is that **the charger never receives the PIN, nor does it validate the User ID in any fashion**. This means that **a malicious user may craft a valid authentication packet using only the serial number of the device**, with any random User ID that the attacker wishes. To make matters worse, the serial number is broadcast over Bluetooth to aid with connection, which means that **chargers can be connected to _and authenticated with_ simply by being within bluetooth range**.

We theorise that the charger has no knowledge of the PIN to perform authentication (as an internet connection cannot be guaranteed, especially during setup), and that the User ID is only transmitted for the purposes of audit logging on the charger.

## Disclosure

The discoverers of this vulnerability are committed to responsible disclosure.

There was a substantial delay of over a year and a half between discovery of this vulnerability
and successful disclosure to the manufacturer.
Gaps between contact attempts were almost universally down to life events.
No public usage of this vulnerability is known to the authors between discovery and public disclosure.

#### Disclosure Timeline

- 23/10/2021: Email sent to ABB
  - **No response**
- 15/02/2022: Called ABB Contact Centre UK
  - Redirected to ABB Contact Centre CH
  - Unable to help, said that email was the only option for contact
- 15/02/2022: Second email sent to ABB
  - **No response**
- 25/03/2022: Phone call to high-profile user of Terra AC
  - Callback arranged for 28th, **no response**
- 31/03/2022: Follow-up phonecall to high-profile user of Terra AC
  - Email address left, **no response**
- 15/04/2022: Third email sent to ABB
  - **No response**
- 21/04/2022: Fourth email sent to ABB from different origin email address
  - **No response**
- 21/10/2022: Contact made with journalist at _Ars Technica_
  - Journalist attempts to make contact with ABB
- 31/10/2022: Journalist still attempting to contact ABB
- 20/01/2023: Journalist achieves contact with ABB, provides contact details
- 26/01/2023: Email sent 26/01 to ABB contact (delay due to life events)
  - Acknowledgement same day
- 30/01/2023: Contact from ABB Cybersecurity Team (_finally!_)
  - Initial disclosure made
- 01/01/2023: **Full disclosure made**
- 03/02/2023: **Manufacturer acknowledges vulnerability** pending investigation
- 16/02/2023: CVEs reserved, further detail given
- 02/03/2023: Progress update
- 24/04/2023: Progress update; patch in testing, secondary party separately reports vulnerability
- 10/05/2023: **Patch released** to ABB partners
- 17/05/2023: **Patch released to public**, public security advisory published by ABB
- 10/07/2023: **Public disclosure**, PoC release