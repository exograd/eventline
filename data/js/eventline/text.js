"use strict";

function evSentence(text) {
  if (text.length === 0) {
    return;
  }

  const firstChar = text.charAt(0);
  const lastChar = text.charAt(text.length-1);

  const suffix = ".!?".includes(lastChar) ? "" : ".";

  return firstChar.toLocaleUpperCase() + text.slice(1) + suffix;
}
