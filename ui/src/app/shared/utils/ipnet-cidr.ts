import { AbstractControl, ValidationErrors, ValidatorFn } from "@angular/forms";
import { IPNet } from "@shared/interfaces";

export function ipNetToCidr(ipNet: IPNet): string {
  const ip = ipNet.IP;

  // Decode the base64 mask and convert it to a binary string
  const maskBase64 = ipNet.Mask;
  const maskBinaryStr = atob(maskBase64)
      .split('')
      .map(char => char.charCodeAt(0).toString(2).padStart(8, '0'))
      .join('');

  // Count the number of '1's in the binary string to get the mask length
  const maskLength = maskBinaryStr.split('1').length - 1;

  return `${ip}/${maskLength}`;
}

export function cidrToIpNet(cidr: string): IPNet {
  const [ip, maskLengthStr] = cidr.split('/');
  const maskLength = parseInt(maskLengthStr, 10);

  // Create a binary string of mask length '1's followed by '0's to make a length of 32
  const maskBinaryStr = '1'.repeat(maskLength) + '0'.repeat(32 - maskLength);

  // Convert the binary string to bytes and then to a base64 string
  const maskBytes = [];
  for (let i = 0; i < maskBinaryStr.length; i += 8) {
    const byte = maskBinaryStr.substring(i, i + 8);
    maskBytes.push(parseInt(byte, 2));
  }
  const maskBase64 = btoa(String.fromCharCode(...maskBytes));

  return { IP: ip, Mask: maskBase64 };
}

export function cidrValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    const cidrRegex = /^([0-9]{1,3}\.){3}[0-9]{1,3}\/([0-9]|[1-2][0-9]|3[0-2])$/;

    // Check if the field is empty
    if (!control.value) {
      return { 'required': true };
    }

    // Check if the value matches the CIDR format
    if (!cidrRegex.test(control.value)) {
      return { 'invalidCidr': { value: control.value } };
    }

    return null;
  };
}
