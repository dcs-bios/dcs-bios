The Dashboard
=============

The Dashboard is the first screen you see when you open the web interface.

Status Indicators
-----------------

At the top, you can find various status indicators:

.. image:: images/dashboard-status-indicators.png


* **Virtual Cockpit** indicates that the DCS-BIOS Hub is exchanging cockpit data and commands with DCS: World.
* The **Lua Console** indicator shows that DCS: World is ready to accept arbitrary snippets of Lua code from the DCS-BIOS Hub.


The other two indicators show the current state of two settings that you can toggle from the system tray menu:

.. image:: images/systray-both-checked.png

* When **Enable access over the network** is checked, the web interface is accessible from other computers on the network. This can be helpful if you need to use the web interface while flying and it's more convenient to use an external device rather than switching between DCS: World and a web browser on the same machine.
* When **Enable Lua Console** is checked and the Lua Console has been set up on the :doc:`DCS Connection<dcs-connection>` screen, you can use the web interface to execute arbitrary snippets of Lua code within DCS.

.. warning::
    Note that if both of these settings are enabled at the same time, anyone who can access TCP port 5010 on your computer can run arbitrary code on your machine. If you do this, make sure your computer is not directly reachable via the internet.


Installed Modules
-----------------

Below the status indicators, you will find shortcuts to the :doc:`control reference documentation <control-reference>` for any installed DCS: World modules that are supported by DCS-BIOS.

.. note:: DCS-BIOS counts a module it knows about as "installed" if it can find a folder of the same name under `mods/aircraft` in either the release or open beta version of DCS: World. This does not work when the folder name differs from the name of the DCS-BIOS module definition, e.g. for the F-18.

Managing Serial Port Connections
--------------------------------

The Dashboard screen displays a list of serial ports and allows you to configure which of these you want the DCS-BIOS Hub to connect to. If you build custom control panels, each Arduino board you connect to your PC will show up as a (virtual) COM port here. You might also see some real RS-232 ports listed, if your main board still has any.
In the following screenshot, COM1 is a real RS-232 port and is not being used, while COM4 and COM6 belong to Arduino boards that are connected via USB.

.. image:: images/dashboard-serial-ports.png

The "Autoconnect" checkbox tells the DCS-BIOS Hub that it should connect to this port when the DCS-BIOS Hub is started or when the COM port "appears", i.e. the device is plugged in and the port shows up as a new device.
If you unplug a device, its COM port will disappear from the list if autoconnect was not checked. If autoconnect was enabled, the port will be listed as "missing" instead and the connection will be reestablished as soon as it appears again.

The individual "Connect" and "Disconnect" buttons on the right can be used to temporarily connect or disconnect a port without changing the autoconnection setting.

The "Disconnect All" button disconnects from all COM ports.

The "Connect All Auto" button connects to all COM ports that have autoconnect enabled.

