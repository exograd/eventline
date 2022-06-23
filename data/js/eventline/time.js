"use strict";

function evDateToUnix(date) {
  return Math.floor(date.getTime() / 1000);
}

function evDayStart(date) {
  const date2 = new Date(date.getTime());
  date2.setUTCHours(0);
  date2.setUTCMinutes(0);
  date2.setUTCSeconds(0);
  date2.setUTCMilliseconds(0);
  return date2;
}

function evNextDayStart(date) {
  const date2 = new Date(date.getTime());
  date2.setDate(date2.getDate() + 1);
  return evDayStart(date2);
}

function evHourStart(date) {
  const date2 = new Date(date.getTime());
  date2.setUTCMinutes(0);
  date2.setUTCSeconds(0);
  date2.setUTCMilliseconds(0);
  return date2;
}

function evNextHourStart(date) {
  const date2 = new Date(date.getTime());
  date2.setHours(date2.getHours() + 1);
  return evHourStart(date2);
}

function evFormatShortDuration(d) {
  if (d == 0) {
    return "00:00";
  }

  d = Math.ceil(d);

  const minutes = Math.floor(d / 60);
  const seconds = d % 60;

  if (minutes == 0) {
    return seconds + "s";
  }

  return minutes+"m" + seconds+"s";
}
