# T E R R A F O R M E R

## what

_Terraformer_ is a utility that acts as a proof-of-concept of the Terra AC Wallbox authentication bypass vulnerability known as BLUTACC (BLUetooth Terra AC wallbox Control).

It has two operating modes:

 * Passing the serial number of a Terra AC Wallbox will open a session with that CP, authenticate, and then set the CP into free-vend mode.
 * Running the command without arguments will continuously loop through all chargers in range, performing the above routine.

There is support in code for a number of additional commands which are not presently exposed via the command line interface.

FYI: This isn't the best written piece of software on the planet - expect bugs and quirks, especially if running against patched wallboxes.
_Please feel free to raise issues and submit PRs!_

## who

_Terraformer_ was written by @duckfullstop.