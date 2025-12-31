import CryptoJS from 'crypto-js';

// This key is used for obfuscation during transmission. 
// In a real-world scenario, this should be exchanged securely or use RSA.
const TRANSMISSION_KEY = CryptoJS.enc.Utf8.parse('miko-transmission-key-32bytes-!!');
const IV = CryptoJS.enc.Utf8.parse('miko-iv-16bytes!');

export const encryptPassword = (password: string): string => {
  const encrypted = CryptoJS.AES.encrypt(password, TRANSMISSION_KEY, {
    iv: IV,
    mode: CryptoJS.mode.CBC,
    padding: CryptoJS.pad.Pkcs7
  });
  return encrypted.toString();
};
