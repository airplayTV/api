// 来源：https://myd04.com/player/js/setting.js?v=4

/**
 * rc4Decode
 * @param data
 * @param _key
 * @param t
 * @returns {string}
 */
function rc4Decode(data, _key, t) {
  let pwd = _key || 'ffsirllq';
  let cipher = '';
  let key = [];
  let box = [];
  let pwd_length = pwd.length;
  if (t === 1) {
    data = atob(data);
  } else {
    data = encodeURIComponent(data);
  }


  let data_length = data.length;
  let i = 0;

  for (let i = 0; i < 256; i++) {
    key[i] = pwd[i % pwd_length].charCodeAt();
    box[i] = i;
  }
  for (let j = i = 0; i < 256; i++) {
    j = (j + box[i] + key[i]) % 256;
    let tmp = box[i];
    box[i] = box[j];
    box[j] = tmp;
  }
  for (let a = j = i = 0; i < data_length; i++) {
    a = (a + 1) % 256;
    j = (j + box[a]) % 256;
    let tmp = box[a];
    box[a] = box[j];
    box[j] = tmp;
    let k = box[((box[a] + box[j]) % 256)];
    cipher += String.fromCharCode(data[i].charCodeAt() ^ k);
  }
  if (t === 1) {
    return decodeURIComponent(cipher);
  } else {
    return btoa(cipher);
  }
}
