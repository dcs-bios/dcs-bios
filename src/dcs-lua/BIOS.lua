dofile(BIOS.LuaScriptDir..[[lib\AircraftList.lua]])

BIOS.dbg = {}
BIOS.logfile = io.open(lfs.writedir()..[[Logs\DCS-BIOS.log]], "w")
function BIOS.log(str) 
	if BIOS.logfile then
		BIOS.logfile:write(str.."\n")
		BIOS.logfile:flush()
	end
end
--in the Plane lua's to log any variables value to the BIOS.log  - BIOS.log(VARIABLE_NAME) ex: BIOS.log(freq)

package.path  = package.path..";.\\LuaSocket\\?.lua"
package.cpath = package.cpath..";.\\LuaSocket\\?.dll"
  
socket = require("socket")

dofile(BIOS.LuaScriptDir..[[lib\Util.lua]])
dofile(BIOS.LuaScriptDir..[[lib\ProtocolIO.lua]])
dofile(BIOS.LuaScriptDir..[[lib\Protocol.lua]])
dofile(BIOS.LuaScriptDir..[[lib\MetadataEnd.lua]])
dofile(BIOS.LuaScriptDir..[[lib\MetadataStart.lua]])
dofile(BIOS.LuaScriptDir..[[lib\CommonData.lua]])
dofile(BIOS.LuaScriptDir..[[lib\A-4E-C.lua]])
dofile(BIOS.LuaScriptDir..[[lib\A10C.lua]])
dofile(BIOS.LuaScriptDir..[[lib\AJS37.lua]])
dofile(BIOS.LuaScriptDir..[[lib\AV8BNA.lua]])
dofile(BIOS.LuaScriptDir..[[lib\Bf109k4.lua]])
dofile(BIOS.LuaScriptDir..[[lib\C-101CC.lua]])
dofile(BIOS.LuaScriptDir..[[lib\ChristenEagle.lua]])
dofile(BIOS.LuaScriptDir..[[lib\F-14B.lua]])
dofile(BIOS.LuaScriptDir..[[lib\F-16C_50.lua]])
dofile(BIOS.LuaScriptDir..[[lib\F-5E-3.lua]])
dofile(BIOS.LuaScriptDir..[[lib\F86.lua]])
dofile(BIOS.LuaScriptDir..[[lib\FA-18C_hornet.lua]])
dofile(BIOS.LuaScriptDir..[[lib\FC3.lua]])
dofile(BIOS.LuaScriptDir..[[lib\FW190A8.lua]])
dofile(BIOS.LuaScriptDir..[[lib\FW190D9.lua]])
dofile(BIOS.LuaScriptDir..[[lib\I-16.lua]])
dofile(BIOS.LuaScriptDir..[[lib\Ka50.lua]])
dofile(BIOS.LuaScriptDir..[[lib\L-39ZA.lua]])
dofile(BIOS.LuaScriptDir..[[lib\M2000C.lua]])
dofile(BIOS.LuaScriptDir..[[lib\MB-339PAN.lua]])
dofile(BIOS.LuaScriptDir..[[lib\Mi8MT.lua]])
dofile(BIOS.LuaScriptDir..[[lib\Mig15.lua]])
dofile(BIOS.LuaScriptDir..[[lib\Mig19.lua]])
dofile(BIOS.LuaScriptDir..[[lib\Mig21.lua]])
dofile(BIOS.LuaScriptDir..[[lib\NS430.lua]])
dofile(BIOS.LuaScriptDir..[[lib\P-51D.lua]])
dofile(BIOS.LuaScriptDir..[[lib\SA342M.lua]])
dofile(BIOS.LuaScriptDir..[[lib\SpitfireLFMkIX.lua]])
dofile(BIOS.LuaScriptDir..[[lib\UH1H.lua]])
dofile(BIOS.LuaScriptDir..[[lib\Yak-52.lua]])
dofile(BIOS.LuaScriptDir..[[BIOSConfig.lua]])

-- Prev Export functions.
local PrevExport = {}
PrevExport.LuaExportStart = LuaExportStart
PrevExport.LuaExportStop = LuaExportStop
PrevExport.LuaExportBeforeNextFrame = LuaExportBeforeNextFrame
PrevExport.LuaExportAfterNextFrame = LuaExportAfterNextFrame

-- Lua Export Functions
LuaExportStart = function()
	
	for _, v in pairs(BIOS.protocol_io.connections) do v:init() end
	BIOS.protocol.init()
	
	-- Chain previously-included export as necessary
	if PrevExport.LuaExportStart then
		PrevExport.LuaExportStart()
	end
end

LuaExportStop = function()
	
	for _, v in pairs(BIOS.protocol_io.connections) do v:close() end
	
	-- Chain previously-included export as necessary
	if PrevExport.LuaExportStop then
		PrevExport.LuaExportStop()
	end
end

function LuaExportBeforeNextFrame()
	
	for _, v in pairs(BIOS.protocol_io.connections) do
		if v.step then v:step() end
	end
	
	-- Chain previously-included export as necessary
	if PrevExport.LuaExportBeforeNextFrame then
		PrevExport.LuaExportBeforeNextFrame()
	end
	
end

function LuaExportAfterNextFrame()
	
	BIOS.protocol.step()
	BIOS.protocol_io.flush()

	-- Chain previously-included export as necessary
	if PrevExport.LuaExportAfterNextFrame then
		PrevExport.LuaExportAfterNextFrame()
	end
end