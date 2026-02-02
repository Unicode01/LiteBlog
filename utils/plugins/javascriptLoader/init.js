function buildArg(name, data, type) {
    if (!type) {
        type = "string"
    }
    switch (type) {
        case "string":
            data = String(data);
            break;
        case "int":
            data = parseInt(data);
            break;
        case "json":
            data = JSON.stringify(data);
            break;
    }
    return {
        Name: name,
        Type: type,
        Data: data
    }
}

function parseArg(arg) {
    return arg;
}

function parseArgs(args) {
    const result = {};
    args.forEach(element => {
        switch (element.Type) {
            case "json":
                // JSON 类型的 Data 现在在 Go 端已转换为字符串，直接解析即可
                try {
                    if (typeof element.Data === 'string') {
                        element.Data = JSON.parse(element.Data);
                    } else if (typeof element.Data === 'object' && element.Data !== null) {
                        // 已经是对象，不需要解析（向后兼容）
                        element.Data = element.Data;
                    } else {
                        element.Data = JSON.parse(String(element.Data));
                    }
                } catch (e) {
                    log(3, "JSON解析失败: " + element.Name);
                    throw e;
                }
                break;
            case "int":
                element.Data = parseInt(element.Data);
                break;
            case "string":
                element.Data = element.Data;
                break;

        }
        result[element.Name] = element.Data;
    });
    return result;
}

// convert uint8array to string
function bytesToString(data) {
    let result = '';
    let i = 0;
    while (i < data.length) {
        let byte1 = data[i++];

        // 1-byte sequence (ASCII)
        if (byte1 < 0x80) { // 0xxxxxxx
            result += String.fromCharCode(byte1);
        }
        // 2-byte sequence
        else if ((byte1 & 0xE0) === 0xC0) { // 110xxxxx
            let byte2 = data[i++];
            if ((byte2 & 0xC0) !== 0x80) {
                // Invalid sequence, add replacement character
                result += '';
                continue;
            }
            const codePoint = ((byte1 & 0x1F) << 6) | (byte2 & 0x3F);
            result += String.fromCodePoint(codePoint);
        }
        // 3-byte sequence
        else if ((byte1 & 0xF0) === 0xE0) { // 1110xxxx
            let byte2 = data[i++];
            let byte3 = data[i++];
            if ((byte2 & 0xC0) !== 0x80 || (byte3 & 0xC0) !== 0x80) {
                result += '';
                continue;
            }
            const codePoint = ((byte1 & 0x0F) << 12) | ((byte2 & 0x3F) << 6) | (byte3 & 0x3F);
            result += String.fromCodePoint(codePoint);
        }
        // 4-byte sequence
        else if ((byte1 & 0xF8) === 0xF0) { // 11110xxx
            let byte2 = data[i++];
            let byte3 = data[i++];
            let byte4 = data[i++];
            if ((byte2 & 0xC0) !== 0x80 || (byte3 & 0xC0) !== 0x80 || (byte4 & 0xC0) !== 0x80) {
                result += '';
                continue;
            }
            const codePoint = ((byte1 & 0x07) << 18) | ((byte2 & 0x3F) << 12) | ((byte3 & 0x3F) << 6) | (byte4 & 0x3F);
            result += String.fromCodePoint(codePoint);
        }
        // Invalid starting byte
        else {
            result += '';
        }
    }
    return result;
}

function genNamespace() {
    const randomStr = Math.random().toString(36).slice(2);
    if (/^\d/.test(randomStr)) {
        const randomLetter = String.fromCharCode(97 + Math.floor(Math.random() * 26)); // 生成a-z随机字母
        return randomLetter + randomStr.slice(1);
    }
    return randomStr;
}

function injectNamespace(namespace, name, func) {
    newname = namespace + "_" + name;
    // build new function
    const newfunc = func
    // add link to original function
    this[newname] = newfunc;
    return newname;
}
