# psminimize
psminimize is a simple utility that tries to minimize a powershell script file. It only uses basic logic to perform the minimization but in general can reduce a ps1 file by half depending on the variable name length. As this is really just a fancy find and replace there are some edge cases to watch out for. 

## Limitations
* Function parameter variables will be renamed. If you define the function/cmdlet within the script and rely on it's name when calling later you will need to manually fix the calling statement.

## Usage
`psminimize -s script.ps1 -o script.min.ps`

|long|short|description|required|
|----|----|----|----|
|script-path|s|The path to the script file to minimize.|true|
|output-path|o|The path to write the script two.|true|



