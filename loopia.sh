#!/bin/sh

request_zone_records(){
    cat<<EOF
<?xml version="1.0" encoding="UTF-8"?>
<methodCall>
    <methodName>getZoneRecords</methodName>
    <params>
        <param><string>$LOOPIA_USER</string></param>
        <param><string>$LOOPIA_PASSWORD</string></param>
        <param><string>$LOOPIA_CUSTOMERNO</string></param>
        <param><string>$ZONE</string></param>
        <param><string>$1</string></param>
    </params>
</methodCall>
EOF
}

request_domains() {
    cat<<EOF
<?xml version="1.0" encoding="UTF-8"?>
<methodCall>
    <methodName>getSubdomains</methodName>
    <params>
        <param><string>$LOOPIA_USER</string></param>
        <param><string>$LOOPIA_PASSWORD</string></param>
        <param><string>$LOOPIA_CUSTOMERNO</string></param>
        <param><string>$ZONE</string></param>
    </params>
</methodCall>
EOF
}


method=$1
shift 1
echo ""
case $method in
    getZoneRecords)
        curl -X POST https://api.loopia.se/RPCSERV -H "Content-Type: application/xml" -d "$(request_zone_records $1)"
        ;;
    getSubdomains)
        curl -X POST https://api.loopia.se/RPCSERV -H "Content-Type: application/xml" -d "$(request_domains $1)"
        ;;
    *)
        echo -n "unknown"
        ;;
esac
echo ""