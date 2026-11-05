import QtQuick

QtObject {
    property string status: "disconnected"
    function connect() { status = "connecting" }
}
