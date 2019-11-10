Hub Scripts
===========

Hub scripts are scripts written in the `Lua programming language <https://www.lua.org/manual/5.1/>`_ that are executed by the DCS-BIOS Hub.
A hub script can read data that is coming from DCS ("sim data") and override data that is sent to the serial ports ("panel data").
It can also intercept commands being sent to the DCS-BIOS Hub and send commands to DCS.

Hub scripts are mostly used to make a physical cockpit that was built for a specific airframe work with other DCS: World modules.

Lifecycle
---------

On the :doc:`Dashboard <dashboard>` screen, you can configure a list of hub scripts. These scripts are executed when the DCS-BIOS Hub starts.
When the "Reload Scripts" button is clicked, the Lua state is thrown away, a new Lua state is created and all hub scripts are executed again
(that means no data survives a click of the "Reload Scripts" button).

Lua Environment
---------------

The DCS-BIOS Hub is using `gopher-lua <https://github.com/yuin/gopher-lua>`_, an implementation of Lua 5.1 in the Go programming language.
Lua 5.1 is the same version that DCS: World uses for its Lua scripts. However, as it is written in Go, you won't be able to load Lua libraries written in C++.
See also: `Differences between Lua and GopherLua <https://github.com/yuin/gopher-lua#differences-between-lua-and-gopherlua>`_

.. highlight:: lua

All hub scripts are executed within the same Lua state, but each hub script is loaded in its own global environment.
That means you can use global variables in your hub script without worrying about conflicts with other hub scripts.

For debugging, you can use the :doc:`Lua Console <lua-console>` and select the "hub" environment. To execute code in the global environment of
a specific hub script, use the *enterEnv* function like this::

    enterEnv("myscript.lua") -- myscript.lua must be a suffix of the script path
    -- any code after the enterEnv line will be executed in the global Lua environment
    -- of the first hub script whose path ends with "myscript.lua" (case insensitive match).

    return MyGlobalVariable -- inspect the value of MyGlobalVariable

The DCS-BIOS Hub provides a few useful functions in the "hub" Lua module, which is provided automatically in the global variable "hub".
The following sections will describe these functions.

Registering Callbacks
---------------------

An **output callback** is a function that takes no output parameters and will be called whenever new data from DCS has been received.
Use *hub.registerOutputCallback()* to create an output callback:

.. code::

    hub.registerOutputCallback(function()
        -- use hub.getSimString() and hub.getSimInteger() here
        -- to access data from the sim
    end)

An **input callback** is a function receiving two arguments (*cmd* and *arg*) and returning a boolean.
It is called whenever a command is received via a serial port.

If the callback function returns *true*, the command is not forwarded to DCS.

An input callback will typically remap commands by using *hub.sendSimCommand()* to send a different command and then returning true to prevent
the original command from being sent to DCS::

    -- remap the "UFC_B1" command from a Harrier cockpit to "UFC_1" in the Hornet: 
    hub.registerInputCallback(function(cmd, arg)
        local acftName = hub.getSimString("MetadataStart/_ACFT_NAME")
        if acftName == "FA-18C_hornet" then
            if cmd == "UFC_B1" then
                hub.sendSimCommand("UFC_1", arg)
                return true
            end
        end
    end)


.. note::

    Output callbacks are executed in the order they were registered. An output callback that has been registered later can overwrite panel data that was set by callbacks that were registered earlier.

    Input callbacks are executed in the *reverse* order they were registered. If one input callback returns true, the remaining ones will not be called.
    An input callback that has been registered later can intercept a command before callbacks that were registered earlier get the chance.

    This means that when multiple hub scripts want to set the same output value or intercept the same command, the script that is last in the list always wins.

Sending Commands to DCS: World
------------------------------

You can send a command to DCS: World using the *hub.sendSimCommand* function.
For example, to click the master caution button in the A-10C::

    hub.sendSimCommand("UFC_MASTER_CAUTION", "1")
    hub.sendSimCommand("UFC_MASTER_CAUTION", "0")

The channel for commands to DCS has a buffer size of 10, so you can send small sequences like pushing and releasing a button
without worrying about blocking anything.

Reading Data from DCS: World
----------------------------

You can access the most recent data that was received from DCS: World with the *getSimString* and *getSimInteger* functions.
They take a control identifier of the form *AircraftName/ElementName* and return a string or integer value. If the control identifier is invalid,
*getSimInteger* will return -1 and *getSimString* will return the empty string.

Note that calling these functions with control identifiers that do not belong to the currently active aircraft in DCS: World will result in undefined behavior (returning garbage data).

Refer to the next section for an example that uses the *getSimString* function.

Overriding Panel Data
---------------------

The DCS-BIOS Hub keeps two copies of export data. One is the *Sim Data* buffer which contains the most recent cockpit state received from DCS: World.
The other is the *Panel Data* buffer which contains the data that is sent to the serial ports.

When receiving new data from DCS: World, the following steps are executed:

* Copy the *Sim Data* buffer to the *Panel Data* buffer
* Execute all output callback functions
* Send the current state of the *Panel Data* buffer to the serial ports

The functions *hub.setPanelInteger* and *hub.setPanelString* can be used to overwrite data in the *Panel Buffer*.
The first parameter is a control identifier and the second is the new value.

For example, the following output callback will display the F-18C Hornet's UFC data on a simpit that was built for the AV8BNA Harrier::

    local function remapOutput(a, b)
        hub.setPanelString(b, hub.getSimString(a))
    end

    hub.registerOutputCallback(function()
        local acftName = getSimString("MetadataStart/_ACFT_NAME")
        if acftName == "FA-18C_hornet" then
            remapOutput("FA-18C_hornet/UFC_COMM1_DISPLAY", "AV8BNA/UFC_COMM1_DISPLAY")
            remapOutput("FA-18C_hornet/UFC_COMM2_DISPLAY", "AV8BNA/UFC_COMM2_DISPLAY")
            local scratchpad = getSimString("FA-18C_hornet/UFC_SCRATCHPAD_STRING_1_DISPLAY")
            scratchpad = scratchpad .. getSimString("FA-18C_hornet/UFC_SCRATCHPAD_STRING_2_DISPLAY")
            scratchpad = scratchpad .. getSimString("FA-18C_hornet/UFC_SCRATCHPAD_NUMBER_DISPLAY")
            setPanelString("AV8BNA/UFC_SCRATCHPAD", scratchpad)

            remapOutput("FA-18C_hornet/UFC_OPTION_CUEING_1", "AV8BNA/AV8BNA_ODU_1_SELECT")
            remapOutput("FA-18C_hornet/UFC_OPTION_DISPLAY_1", "AV8BNA/AV8BNA_ODU_1_Text")
            remapOutput("FA-18C_hornet/UFC_OPTION_CUEING_2", "AV8BNA/AV8BNA_ODU_2_SELECT")
            remapOutput("FA-18C_hornet/UFC_OPTION_DISPLAY_2", "AV8BNA/AV8BNA_ODU_2_Text")
            remapOutput("FA-18C_hornet/UFC_OPTION_CUEING_3", "AV8BNA/AV8BNA_ODU_3_SELECT")
            remapOutput("FA-18C_hornet/UFC_OPTION_DISPLAY_3", "AV8BNA/AV8BNA_ODU_3_Text")
            remapOutput("FA-18C_hornet/UFC_OPTION_CUEING_4", "AV8BNA/AV8BNA_ODU_4_SELECT")
            remapOutput("FA-18C_hornet/UFC_OPTION_DISPLAY_4", "AV8BNA/AV8BNA_ODU_4_Text")
            remapOutput("FA-18C_hornet/UFC_OPTION_CUEING_5", "AV8BNA/AV8BNA_ODU_5_SELECT")
            remapOutput("FA-18C_hornet/UFC_OPTION_DISPLAY_5", "AV8BNA/AV8BNA_ODU_5_Text")
        end
    end)

