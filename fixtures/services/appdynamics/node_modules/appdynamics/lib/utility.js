function filterSensitiveDataFromObject(objIns) {
  if (objIns && Object.prototype.toString.call(objIns) == "[object Object]") {
    Object.keys(objIns).forEach(function (key) {
      objIns[key] = filterSensitiveDataFromObject(objIns[key]);
    });
    return objIns;
  }
  return "?";
}

function deepCopy(origObject) {
  let cpObj, value, key;
  if (typeof origObject !== "object" || origObject === null) {
    return origObject;
  }
  cpObj = Array.isArray(origObject) ? [] : {};
  for (key in origObject) {
    value = origObject[key];
    cpObj[key] = deepCopy(value);
  }
  return cpObj;
}

module.exports.filterSensitiveDataFromObject = filterSensitiveDataFromObject;
module.exports.deepCopy = deepCopy;