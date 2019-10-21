package dcssetup

import "os"

func getHookDefinition(name string) *hookDefinition {
	if name == "autostart" {
		executable, err := os.Executable()
		if err != nil {
			return nil
		}
		return &hookDefinition{
			filename: "DCS-BIOS-Autostart-hook.lua",
			content: `net.log("Starting DCS-BIOS Hub")
			require('os').run_process([[` + executable + `]], "--autorun-mode")
			`,
		}
	} else if name == "luaconsole" {
		return &hookDefinition{
			filename: "DCS-BIOS-LuaConsole-hook.lua",
			content: `net.log("loading Lua Console GameGUI.")

			local require = require
			local loadfile = loadfile
			
			package.path = package.path..";.\\LuaSocket\\?.lua"
			package.cpath = package.cpath..";.\\LuaSocket\\?.dll"
			
			local JSON = loadfile("Scripts\\JSON.lua")()
			local socket = require("socket")
			
			local function runSnippetIn(env, code)
				
				local resultStringCode = [[
			local function serialize(svalue)
				local seenTables = {}
				local retlist = {}
				local indentLevel = 0
				local function serializeRecursive(value)
					if type(value) == "string" then return table.insert(retlist, string.format("%q", value)) end
					if type(value) ~= "table" then return table.insert(retlist, tostring(value)) end
						
					if seenTables[value] == true then
						   table.insert(retlist, tostring(value))
						return
					end
					seenTables[value] = true
					
					-- we have a table, iterate over the keys
			
					table.insert(retlist, "{\n")
					indentLevel = indentLevel + 4
					for k, v in pairs(value) do
						table.insert(retlist, string.rep(" ", indentLevel).."[")
						if type(k) == "table" then
							   table.insert(retlist, tostring(k))
						else
							serializeRecursive(k)
						end
						table.insert(retlist, "] = ")
						serializeRecursive(v)
						table.insert(retlist, ",\n")
					end
					indentLevel = indentLevel - 4
					table.insert(retlist, string.rep(" ", indentLevel).."}")
				end
				serializeRecursive(svalue, "    ")
				return table.concat(retlist)
			end
			
			
			local function evalAndSerializeResult(code)
				local success = false
				local result = ""
				local retstatus = ""
				
				local f, error_msg = loadstring(code, "Lua Console Snippet")
				if f then
					--setfenv(f, _G)
					success, result = pcall(f)
					if success then
						retstatus="success"
						result = serialize(result)
					else
						retstatus = "runtime_error"
						result = tostring(result)
					end
				else
					retstatus = "syntax_error"
					result = tostring(error_msg)
				end
				
				return retstatus.."\n"..result
			end
				
				]].."return evalAndSerializeResult("..string.format("%q", code)..")"
				local result = nil
				local success = nil
				
				if env == "gui" then
					result = loadstring(resultStringCode)()
					success = true
				else
					result, success = net.dostring_in(env, resultStringCode)
					log.write("Lua Console", log.INFO, "l94: success="..tostring(success))
				end
				
				if not success then
					result = "dostring_error\n"..tostring(env).."\n"..tostring(code).."\n"..tostring(result)
				end
				
				local firstNewlinePos = string.find(result, "\n")
				log.write("Lua Console", log.INFO, "firstnewlinepos="..tostring(firstNewlinePos))
				
				local result_str = string.sub(result, firstNewlinePos+1)
				local status_str = string.sub(result, 1, firstNewlinePos-1)
				return result_str, status_str
			end
			
			
			
			witchcraft = {}
			
			witchcraft.host = "localhost"
			witchcraft.port = 3001
			
			witchcraft.state = "closed"
			witchcraft.timeClosed = 0
			witchcraft.timeOfLastSendAttempt = 0
			
			local function reconnect()
				log.write('Lua Console', log.INFO, "attempting to connect at real time "..tostring(DCS.getRealTime()))
				if witchcraft.conn ~= nil then
					witchcraft.conn:close()
				end
			
				witchcraft.state = "connecting"
				witchcraft.txbuf = '{"type":"ping"}\n'
				witchcraft.rxbuf = ""
				witchcraft.conn = socket.tcp()
				witchcraft.conn:settimeout(.0001)
				witchcraft.conn:connect(witchcraft.host, witchcraft.port)
			end
			
			
			local function step()
				if witchcraft.state == "closed" then
					if DCS.getRealTime() - witchcraft.timeClosed > 2 then
						reconnect()
					end
				end
				
				--if witchcraft.txbuf == "" then
				--	witchcraft.txbuf = witchcraft.txbuf .. '{"type":"ping"}\n'
				--else
				--	--log.write("Lua Console", log.INFO, "txbuf has length "..tostring(string.len(witchcraft.txbuf)))
				--end
				if witchcraft.txbuf:len() > 0 then
					witchcraft.timeOfLastSendAttempt = DCS.getRealTime()
					local bytes_sent = nil
					--local ret1, ret2, ret3 = witchcraft.conn:send(witchcraft.txbuf)
					local bytes_sent, err_msg, err_bytes_sent = witchcraft.conn:send(witchcraft.txbuf)
					if bytes_sent == nil then
						--env.info("could not send witchcraft: "..ret2)
						if err_bytes_sent == 0 then
							if err_msg == "closed" then
			--					witchcraft.txbuf = '{"type":"ping"}\n'
			--					witchcraft.rxbuf = ""
			--					witchcraft.lastUnitUpdateTime = 0
			--					witchcraft.conn = socket.tcp()
			--					witchcraft.conn:settimeout(.0001)
								log.write("Lua Console", log.INFO, "socket was closed")
								witchcraft.state = "closed"
							end
							--env.info("reconnecting to "..tostring(witchcraft.host)..":"..tostring(witchcraft.port))
							witchcraft.conn:connect(witchcraft.host, witchcraft.port)
							return
						end
						bytes_sent = err_bytes_sent
					end
					witchcraft.txbuf = witchcraft.txbuf:sub(bytes_sent + 1)
				else
					if DCS.getRealTime() - witchcraft.timeOfLastSendAttempt > 2 then
						witchcraft.txbuf = '{"type":"ping"}\n'
					end
				end
				
				local line, err = witchcraft.conn:receive()
				if err then
					--env.info("witchcraft read error: "..err)
				else
					msg = JSON:decode(line)
					if msg.type == "lua" then
						local response_msg = {}
						response_msg.type = "luaresult"
						response_msg.name = msg.name
						
						--log.write('LuaConsole', log.INFO, "executing snippet "..msg.code.." in "..msg.luaenv)
						
						if not msg.luaenv then msg["luaenv"] = "export" end
						local result, status = runSnippetIn(msg.luaenv, msg.code)
						--local result, status = "44", "ok"
						
						response_msg.result = tostring(result)
						response_msg.status = status
						
						
						
						local response_string = ""
						local function encode_response()
							response_string = JSON:encode(response_msg):gsub("\n","").."\n"
						end
						
						local success, result = pcall(encode_response)
						if not success then
							response_msg.status = "encode_response_error"
							response_msg.result = tostring(result)
							encode_response()
						end
						
						--log.write("Lua Console", log.INFO, "response_string is "..tostring(response_string))
						--log.write("Lua Console", log.INFO, "response_string has length "..tostring(string.len(response_string)))
						
						witchcraft.txbuf = witchcraft.txbuf .. response_string
					end
				end
					
			
			end
			
			local function stepOld()
				--log.write('Lua Console', log.INFO, 'step()')
				local timeSinceLastPing = DCS.getRealTime() - (witchcraft.lastPingTime or 0)
				--log.write('Lua Console', log.INFO, "time since last ping is "..tostring(timeSinceLastPing).." and time is "..tostring(DCS.getRealTime()))
				if timeSinceLastPing > 2 then
					log.write('Lua Console', log.INFO, "calling reconnect()")
					witchcraft.lastPingTime = DCS.getRealTime()
					reconnect()
				end
				
				if witchcraft.txbuf:len() > 0 then
					local bytes_sent = nil
					local ret1, ret2, ret3 = witchcraft.conn:send(witchcraft.txbuf)
					if ret1 then
						bytes_sent = ret1
					else
						--env.info("could not send witchcraft: "..ret2)
						if ret3 == 0 then
							if ret2 == "closed" then
								reconnect()
							end
							--env.info("reconnecting to "..tostring(witchcraft.host)..":"..tostring(witchcraft.port))
							witchcraft.conn:connect(witchcraft.host, witchcraft.port)
							return
						end
						bytes_sent = ret3
					end
					witchcraft.txbuf = witchcraft.txbuf:sub(bytes_sent + 1)
				else
					if witchcraft.txidle_hook then
						local bool, err = pcall(witchcraft.txidle_hook)
						if not bool then
							--env.info("witchcraft.txidle_hook() failed: "..err)
						end
					end
				end
				
				local line, err = witchcraft.conn:receive()
				if err then
					log.write("Lua Console", log.INFO, "witchcraft read error: "..tostring(err))
				else
					log.write('Lua Console', log.INFO, "got line "..tostring(line))
					msg = JSON:decode(line)
					if msg.type == "ping" then
						witchcraft.lastPingTime = DCS.getRealTime()
						log.write('Lua Console', log.INFO, "got ping at "..tostring(DCS.getRealTime()))
					end
					if msg.type == "lua" then
						local response_msg = {}
						response_msg.type = "luaresult"
						response_msg.name = msg.name
						
						--log.write('LuaConsole', log.INFO, "executing snippet "..msg.code.." in "..msg.luaenv)
						
						if not msg.luaenv then msg["luaenv"] = "export" end
						local result, status = runSnippetIn(msg.luaenv, msg.code)
						--local result, status = "44", "ok"
						
						response_msg.result = tostring(result)
						response_msg.status = status
						
						
						
						local response_string = ""
						local function encode_response()
							response_string = JSON:encode(response_msg):gsub("\n","").."\n"
						end
						
						local success, result = pcall(encode_response)
						if not success then
							response_msg.status = "encode_response_error"
							response_msg.result = tostring(result)
							encode_response()
						end
						
						--log.write("Lua Console", log.INFO, "response_string is "..tostring(response_string))
						log.write("Lua Console", log.INFO, "response_string has length "..tostring(string.len(response_string)))
						
						witchcraft.txbuf = witchcraft.txbuf .. response_string
					end
				end
				
				
			end
			
			
			DCS.setUserCallbacks({
				["onSimulationFrame"] = function()
					status, err = pcall(step)
					if not status then
						log.write("Lua Console Error", log.INFO, tostring(err))
					end
				end
			})
			
			
			log.write('Lua Console', log.INFO, "loaded")
			`,
		}
	}
	return nil
}
