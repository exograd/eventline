"use strict";

function evParseJSONPointer(string) {
  if (string.length === 0) {
    return [];
  }

  if (string[0] !== "/") {
    throw new Error("json pointers must start with '/'");
  }

  const segments = string.split("/").slice(1)

  return segments.map(evDecodeJSONPointerSegment);
}

function evFormatJSONPointer(pointer) {
  return pointer.map(s => {
    return "/" + evEncodeJSONPointerSegment(s)
  }).join("");
}

function evEncodeJSONPointerSegment(segment) {
  return segment.replaceAll("~", "~0").replaceAll("/", "~1");
}

function evDecodeJSONPointerSegment(segment) {
  return segment.replaceAll("~1", "/").replaceAll("~0", "~");
}
