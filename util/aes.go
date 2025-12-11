package util

// 参考：https://www.zhangshengrong.com/p/RmNP8R3PNk/

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
)

//加密过程：
//  1、处理数据，对数据进行填充，采用PKCS7（当密钥长度不够时，缺几位补几个几）的方式。
//  2、对数据进行加密，采用AES加密方法中CBC加密模式
//  3、对得到的加密数据，进行base64加密，得到字符串
// 解密过程相反

//16,24,32位字符串的话，分别对应AES-128，AES-192，AES-256 加密方法
//key不能泄露

// pkcs7Padding 填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(padding)}复制padding个
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	return data[:(length - unPadding)], nil
}

// AesEncrypt 加密
func AesEncrypt(data []byte, key, iv []byte) ([]byte, error) {
	//创建加密实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//判断加密快的大小
	blockSize := block.BlockSize()
	//填充
	encryptBytes := pkcs7Padding(data, blockSize)
	//初始化加密数据接收切片
	crypted := make([]byte, len(encryptBytes))
	//使用cbc加密模式
	blockMode := cipher.NewCBCEncrypter(block, iv[:blockSize])
	//执行加密
	blockMode.CryptBlocks(crypted, encryptBytes)
	return crypted, nil
}

// AesDecrypt 解密
func AesDecrypt(data []byte, key, iv []byte) ([]byte, error) {
	//创建实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块的大小
	blockSize := block.BlockSize()
	//使用cbc
	blockMode := cipher.NewCBCDecrypter(block, iv[:blockSize]) //
	//初始化解密数据接收切片
	crypted := make([]byte, len(data))
	//执行解密
	blockMode.CryptBlocks(crypted, data)
	//去除填充
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return nil, err
	}
	return crypted, nil
}

func AesDecrypt_QiGeJieXi(encryptText string, key []byte) ([]byte, error) {
	// AI: JavaScript转golang

	// 2. Base64 解码 (对应 JS: CryptoJS.enc.Base64.parse)
	ciphertextBytes, err := base64.StdEncoding.DecodeString(encryptText)
	if err != nil {
		return nil, err
	}
	// 校验长度，至少要包含 IV (16字节)
	if len(ciphertextBytes) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	// 3. 提取 IV (前16字节) 和 实际密文 (16字节之后)
	// 对应 JS: words.slice(0, 0x4) 和 words.slice(0x4)
	iv := ciphertextBytes[:aes.BlockSize]
	actualCiphertext := ciphertextBytes[aes.BlockSize:]

	// 4. 创建 AES Cipher Block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 5. 设置 CBC 模式
	if len(actualCiphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	// 6. 解密
	// 注意：CryptBlocks 会直接修改传入的 slice，这里可以直接原地解密
	var plaintext = make([]byte, len(actualCiphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, actualCiphertext)

	// 7. 去除 PKCS7 填充 (CryptoJS 自动处理了，Go 需要手动处理)
	plaintext, err = pkcs7UnPadding(plaintext)
	if err != nil {
		return nil, fmt.Errorf("unpadding error: %v", err)
	}

	// 8. 返回字符串
	return plaintext, nil
}

// EncryptByAes Aes加密 后 base64 再加
func EncryptByAes(key, iv, data []byte) (string, error) {
	res, err := AesEncrypt(data, key, iv)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(res), nil
}

// DecryptByAes Aes 解密
func DecryptByAes(key, iv []byte, data string) ([]byte, error) {
	dataByte, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return AesDecrypt(dataByte, key, iv)
}

// 加密/解密示例
// util.EncryptByAes([]byte("9b0d6d401fc5c57f"), []byte("1234567890983456"), []byte(`md5.enc.Utf8.parse("9b0d6d401fc5c57f").toString()`))
// util.DecryptByAes([]byte("9b0d6d401fc5c57f"), []byte("1234567890983456"), str)
