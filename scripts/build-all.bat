@echo off

echo Please Run Me on the root dir, not in scripts dir.

IF EXIST output (
    echo "Output dir existed, deleting and recreating..."
    rd /s /q output
)
mkdir output\services
pushd src\services
for /D %%i in (*) do (
    if not "%%i"=="health" (
        set "name=%%i"
        setlocal enabledelayedexpansion
        set "capName=!name:~0,1!"
        set "capName=!capName:a=A!"
        set "capName=!capName:b=B!"
	    set "capName=!capName:c=C!"
        set "capName=!capName:d=D!"
        set "capName=!capName:e=E!"
        set "capName=!capName:f=F!"
        set "capName=!capName:g=G!"
        set "capName=!capName:h=H!"
        set "capName=!capName:i=I!"
        set "capName=!capName:j=J!"
        set "capName=!capName:k=K!"
        set "capName=!capName:l=L!"
        set "capName=!capName:m=M!"
        set "capName=!capName:n=N!"
        set "capName=!capName:o=O!"
        set "capName=!capName:p=P!"
        set "capName=!capName:q=Q!"
        set "capName=!capName:r=R!"
        set "capName=!capName:s=S!"
        set "capName=!capName:t=T!"
        set "capName=!capName:u=U!"
        set "capName=!capName:v=V!"
        set "capName=!capName:w=W!"
        set "capName=!capName:x=X!"
        set "capName=!capName:y=Y!"
        set "capName=!capName:z=Z!"
        set "capName=!capName!!name:~1!"
        cd %%i
        go build -o ../../../output/services/%%i/!capName!Service.exe
        cd ..
        endlocal
    )
)


popd
mkdir output\gateway

cd src\web
go build -o ../../output/gateway/Gateway.exe
echo OK!
