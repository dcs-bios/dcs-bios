BIOS.protocol.beginModule("SpitfireLFMkIX", 0x5400)
BIOS.protocol.setExportModuleAircrafts({"SpitfireLFMkIX"})

local documentation = moduleBeingDefined.documentation

local document = BIOS.util.document  

local parse_indication = BIOS.util.parse_indication

local defineFloat = BIOS.util.defineFloat
local defineIndicatorLight = BIOS.util.defineIndicatorLight
local definePushButton = BIOS.util.definePushButton
local definePotentiometer = BIOS.util.definePotentiometer
local defineRotary = BIOS.util.defineRotary
local defineSetCommandTumb = BIOS.util.defineSetCommandTumb
local defineTumb = BIOS.util.defineTumb
local defineToggleSwitch = BIOS.util.defineToggleSwitch
local defineToggleSwitchToggleOnly = BIOS.util.defineToggleSwitchToggleOnly
local defineFixedStepTumb = BIOS.util.defineFixedStepTumb
local defineFixedStepInput = BIOS.util.defineFixedStepInput
local defineVariableStepTumb = BIOS.util.defineVariableStepTumb
local defineString = BIOS.util.defineString
local defineRockerSwitch = BIOS.util.defineRockerSwitch
local defineMultipositionSwitch = BIOS.util.defineMultipositionSwitch
local defineIntegerFromGetter = BIOS.util.defineIntegerFromGetter

-- Oxygen Apparatus Controls
defineTumb("OXY_VALVE", 3, 3003, 13, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.oxygen_valve")  
-- Safety Lever
defineTumb("SAFETY",5, 3001, 3, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.safety")
--Triggers
defineTumb("BUTTON_MG",5, 3003, 4, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.trigger_0")
defineTumb("BUTTON_CAN",5, 3004, 5, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.trigger_1")
defineTumb("BUTTON_LINKED",5, 3005, 6, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.trigger_2")
-- Wheel Brakes Lever
defineTumb("WHEEL_BRAKES",1, 3002, 9, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.wheel_brakes")
-- Main Panel
-- Altimeter
defineRotary("ALT_MBAR",1, 3037, 30, "device_commands", "Cockpit.SpitfireLFMkIX.altimeter")
-- DI
defineRotary("DI",1, 3041, 32, "device_commands", "Cockpit.SpitfireLFMkIX.di")
-- Fuel Gauge Button
defineTumb("FUEL_GAUGE",1, 3005, 44, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.fuel_gauge")
-- Nav. Lights Toggle
defineTumb("NAV_LIGHTS",1, 3007, 46, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.nav_lights")
-- Flaps Lever
defineTumb("FLAPS", 1, 3009, 47, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.flaps")  
-- U/C Indicator Blind
defineTumb("UC_BLIND", 1, 3011, 50, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.uc_blind")  
-- Clock Setter Pinion
defineTumb("CLK_PINION_PULL", 1, 3013, 54, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.clock Pull out.")  
defineRotary("CLK_PINION", 1, 3014, 55, "device_commands", "Cockpit.SpitfireLFMkIX.clock")  
-- Magnetos Toggles
defineTumb("MAGNETO0",2, 3015, 56, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.magneto0")
defineTumb("MAGNETO1",2, 3017, 57, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.magneto1")
-- Supercharger Mode Toggle
defineTumb("BLOWER",2, 3019, 58, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.supercharger_mode")
-- Illumination Controls
definePotentiometer("PITLITE_LH",4, 3001, 60, {0,1}, "device_commands", "Cockpit.SpitfireLFMkIX.illumination_lh")
definePotentiometer("PITLITE_RH",4, 3004, 61, {0,1}, "device_commands", "Cockpit.SpitfireLFMkIX.illumination_rh")
-- Starter Button Cover
defineTumb("STARTER_COVER",2, 3021, 64, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.starter_cover")
-- Starter Button
defineTumb("STARTER",2, 3023, 65, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.starter")
-- Booster Coil Button Cover
defineTumb("BOOSTER_COVER",2, 3025, 66, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.booster_cover")
-- Booster Coil Button
defineTumb("BOOSTER",2, 3027, 67, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.booster")
-- Primer Pump
--TODO definePotentiometer("PRIMER",1, 3030, 69, {0,1}, "device_commands", "Cockpit.SpitfireLFMkIX.primer")
-- Tank Pressurizer Lever
defineTumb("TANK_PRS",2, 3033, 70, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.tank_pressurizer")
-- Magnetic Compass
defineFixedStepTumb("COMPASS_RING", 1, 3017, 74, 0.00333, {0, 1}, {-0.00333, 0.00333}, nil, "device_commands", "Cockpit.SpitfireLFMkIX.compass")
-- Gun Sight and Tertiary Weapons Controls
-- Gun Sight Setter Rings
definePotentiometer("GUNSIGHT_RANGE",5,3007, 77, {0.0, 1.0},"device_commands", "Cockpit.SpitfireLFMkIX.gun_sight_range")
definePotentiometer("GUNSIGHT_BASE",5,3010, 78, {0.0, 1.0},"device_commands", "Cockpit.SpitfireLFMkIX.gun_sight_span")
-- Gun Sight Tint Screen
defineTumb("GUNSIGHT_TINT", 5, 3016, 79, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.gun_sight_tint")
-- Gun Sight Master Switch
defineTumb("GUNSIGHT_SWITCH",5, 3018, 80, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.gun_sight_master")
-- Gun Sight Dimmer
definePotentiometer("GUNSIGHT_DIMMER",5, 3020, 81, {0.0, 1.0}, "device_commands", "Cockpit.SpitfireLFMkIX.gun_sight_illumination")
-- Port Wall
-- Elevator Trim Wheel
definePotentiometer("TRIM_WHEEL",1,3029, 145, {-1.0, 1.0},"device_commands", "Cockpit.SpitfireLFMkIX.trim_elevator")
-- Rudder Trim Wheel
definePotentiometer("RTRIM_WHEEL",1,3044, 146, {-1.0, 1.0},"device_commands", "Cockpit.SpitfireLFMkIX.trim_rudder")
-- Radio Remote Channel Switcher
-- Off Button
defineTumb("RCTRL_OFF", 15, 3001, 115, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_0")
-- A Button
defineTumb("RCTRL_A", 15, 3002, 116, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_a")
-- B Button
defineTumb("RCTRL_B", 15, 3003, 117, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_b")
-- C Button
defineTumb("RCTRL_C", 15, 3004, 118, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_c")
-- D Button
defineTumb("RCTRL_D", 15, 3005, 119, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_d")
-- Dimmer Toggle
defineTumb("RCTRL_DIM",15, 3006, 125, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_dimmer")
-- Transmit Lock Toggle
defineTumb("RCTRL_TLOCK",15, 3017, 155, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_lock")
-- Mode Selector
--TODO NOT WORKING PROPERLY
defineTumb("RCTRL_T_MODE1",15, 3007, 156, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_mode 1")
defineTumb("RCTRL_T_MODE2",15, 3008, 156, 1, {-1,0}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radio_mode 2")
-- Throttle Quadrant
-- Throttle Lever
--TODO elements["THTL"] = default_movable_axis(_("Cockpit.SpitfireLFMkIX.throttle"), devices.ENGINE_CONTROLS, device_commands.Button_3, 126, 0.0, 0.1, true, false)
-- Bomb Drop Button
defineTumb("BUTTON_BOMB",5, 3015, 128, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.trigger_bomb")
-- Airscrew Lever
definePotentiometer("PROP", 2, 3006, 129, {-1.0, 1.0},"device_commands", "Cockpit.SpitfireLFMkIX.pitch")
-- Mix Cut-Off Lever
defineTumb("MIX", 2, 3009, 130, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.mix")
defineTumb("UC_DOWN_C",2, 3099, 131, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.uc_down_indication")
-- Radiator Control Toggle
defineTumb("RADIATOR",1, 3033, 133, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radiator_mode")
-- Pitot Heater Toggle
defineTumb("PITOT",1, 3035, 134, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.pitot")
-- Fuel Pump Toggle
defineTumb("FUEL_PUMP",2, 3043, 135, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.fuel_pump")
-- Carb. Air Control Lever
defineTumb("CARB_AIR", 2, 3045, 137, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.carburettor_flap")
-- Oil Diluter Button Cover
defineTumb("DILUTER_COVER",2, 3051, 157, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.dilution_cover")
-- Oil Diluter Button
defineTumb("DILUTER",2, 3053, 158, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.dilution")
-- Supercharger Mode Test Button Cover
defineTumb("MS_TEST_COVER",2, 3055, 159, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.supercharger_cover")
-- Supercharger Mode Test Button
defineTumb("MS_TEST",2, 3057, 160, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.supercharger_test")
-- Radiator Flap Test Button Cover
defineTumb("RAD_TEST_COVER",2, 3059, 161, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radiator_cover")
-- Radiator Flap Test Button
defineTumb("RAD_TEST",2, 3061, 162, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.radiator_test")
-- Stbd. Wall
-- De-Icer Lever
defineTumb("DEICER", 1, 3021, 87, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.deicer")
-- U/C Emergency Release Lever
defineTumb("UC_EMER", 1, 3023, 88, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.uc_emergency")
-- Wobble Type Fuel Pump
defineTumb("WOBBLE_PUMP", 2, 3035, 90, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.wobble")
-- Morse Key & Apparatus
-- Upward Lamp Mode Selector
defineTumb("MORSE_UP_MODE", 4, 3007, 92, 0.5, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.id_lamp_up_mode")
-- Downward Lamp Mode Selector
defineTumb("MORSE_DN_MODE", 4, 3011, 93,  0.5, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.id_lamp_down_mode")
-- Morse Key
defineTumb("MORSE_KEY",4, 3015, 94, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.morse_key")
-- U/C Lever
defineTumb("UC",1, 3025, 148, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.uc")
-- I.F.F. Control Box
-- I.F.F. Upper Toggle (Type B)
defineTumb("IFF_B",4, 3017, 106, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.iff_b")
-- I.F.F. Lower Toggle (Type D)
defineTumb("IFF_D",4, 3019, 107, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.iff_d")
-- I.F.F. Protective Cover
defineTumb("IFF_COVER",4, 3021, 108, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.iff_cover")
-- I.F.F. Fore Button (0)
defineTumb("IFF_0",4, 3023, 109, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.iff_0")
-- I.F.F. Aft Button (1)
defineTumb("IFF_1",4, 3025, 110, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.iff_1")
-- Fuel Cocks & Tertiary
-- Fuel Cock
defineTumb("FUEL_COCK", 2, 3037, 100, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.fuel_cock_primary")
-- Droptank Cock
defineTumb("DROPTANK_COCK", 2, 3041, 98, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.fuel_cock_droptank")
-- Droptank Release Handle
defineTumb("DROPTANK_JETT", 5, 3041, 99, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.droptank_release")
-- Canopy Controls
-- Cockpit Open/Close Control
defineTumb("HATCH",1, 3051, 149, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.canopy_operate")
-- Cockpit Jettison Pull Ball
defineTumb("HATCH_JETTISON", 1, 3057, 140, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.canopy_jettison")
-- Cockpit Side Door Open/Close Control
defineTumb("SIDE_DOOR",1, 3059, 147, 1, {0,1}, nil, false, "device_commands", "Cockpit.SpitfireLFMkIX.sidedoor_operate")

defineFloat("CANOPY_TRUCKS", 162, {0, 1}, "Indicator", "Canopy_Trucks")
defineFloat("CANOPY_VISIBILITY", 163, {0, 1}, "Indicator", "Canopy_Visibility")
defineFloat("CANOPY_CRANK", 147, {0.0, 1.0}, "Indicator", "Canopy_Crank")
defineFloat("OXYGENDELIVERYGAUGE", 11, {0.0, 0.4}, "Indicator", "OxygenDeliveryGauge")
defineFloat("OXYGENSUPPLYGAUGE", 12, {0.0, 1.0}, "Indicator", "OxygenSupplyGauge")
defineFloat("TRIMGAUGE", 17, {-1.0, 1.0}, "Indicator", "TrimGauge")
defineFloat("PNEUMATICPRESSUREGAUGE", 18, {0.0, 1.0}, "Indicator", "PneumaticPressureGauge")
defineFloat("LEFTWHEELBRAKEGAUGE", 19, {0.0, 1.0}, "Indicator", "LeftWheelBrakeGauge")
defineFloat("RIGHTWHEELBRAKEGAUGE", 20, {0.0, 1.0}, "Indicator", "RightWheelBrakeGauge")
defineFloat("AIRSPEEDGAUGE", 21, {0.0, 0.5}, "Indicator", "AirspeedGauge")
defineFloat("AHORIZONBANK", 23, {-1.0, 1.0}, "Indicator", "AHorizonBank")
defineFloat("AHORIZONPITCH", 24, {-1.0, 1.0}, "Indicator", "AHorizonPitch")
defineFloat("VARIOMETERGAUGE", 25, {-1.0, 1.0}, "Indicator", "VariometerGauge")
defineFloat("ALTIMETERHUNDREDS", 26, {0.0, 1.0}, "Indicator", "AltimeterHundreds")
defineFloat("ALTIMETERTHOUSANDS", 27, {0.0, 1.0}, "Indicator", "AltimeterThousands")
defineFloat("ALTIMETERTENSTHOUSANDS", 28, {0.0, 1.0}, "Indicator", "AltimeterTensThousands")
defineFloat("ALTIMETERSETPRESSURE", 29, {0.0, 1.0}, "Indicator", "AltimeterSetPressure")
defineFloat("DIGAUGE", 31, {0.0, 1.0}, "Indicator", "DIGauge")
defineFloat("SIDESLIPGAUGE", 33, {-1.0, 1.0}, "Indicator", "SideslipGauge")
defineFloat("TURNGAUGE", 34, {-1.0, 1.0}, "Indicator", "TurnGauge")
defineFloat("VOLTMETERGAUGE", 35, {0.0, 1.0}, "Indicator", "VoltmeterGauge")
defineFloat("TACHOMETERGAUGE", 37, {0.0, 0.5}, "Indicator", "TachometerGauge")
defineFloat("BOOSTGAUGE", 39, {0.0, 1.0}, "Indicator", "BoostGauge")
defineFloat("OILPRESSUREGAUGE", 40, {0.0, 1.0}, "Indicator", "OilPressureGauge")
defineFloat("OILTEMPERATUREGAUGE", 41, {0.0, 1.0}, "Indicator", "OilTemperatureGauge")
defineFloat("RADIATORTEMPERATUREGAUGE", 42, {0.0, 0.7}, "Indicator", "RadiatorTemperatureGauge")
defineFloat("FUELRESERVEGAUGE", 43, {0.0, 1.0}, "Indicator", "FuelReserveGauge")

BIOS.protocol.endModule()