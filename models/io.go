// io
package models

import (
	"bufio"
	//"fmt"
	"io"
	"os"
	"regexp"
	//"strconv"
	"strings"
)

func LoadConfig() {
	inputFile, err := os.Open("./root.z")
	if err != nil {
		return
	}
	defer inputFile.Close()
	inputReader := bufio.NewReader(inputFile)
	for {
		inputString, readerError := inputReader.ReadString('\n')
		if readerError == io.EOF {
			return
		}

		req := regexp.MustCompile(`[^\s]+`)
		strs := req.FindAllString(inputString, -1)
		//转为小写
		strs[0] = strings.ToLower(strs[0])
		strs[4] = strings.ToLower(strs[4])
		//fmt.Printf("%+v   %d\n", strs, len(strs))
		//点分格式转化为长度格式 www.baidu.com -> 3www5baidu3com
		//		strs[0] = string(Point_to_len_format([]byte(strs[0])))

		switch strs[3] {
		case "NS":
			//			strs[4] = string(Point_to_len_format([]byte(strs[4])))
			AddNS(strs[0], strs[4])
			break
		case "A":
			AddA_str(strs[0], strs[4])
			break
		case "AAAA":
			AddAAAA_str(strs[0], strs[4])
			break
		case "CNAME":
			SetOriCNAME(strs[0], strs[4])
			break
		case "MX":
			//			SetMX(strs[0], strs[4])
			break
		}

	}

}

func Point_to_len_format(buff []byte) []byte {
	index := 0
	l := 1
	//ret := make([]byte, 0)
	var ret []byte
	ret = append(ret, 0)

	for i := 0; i < len(buff); i++ {
		if buff[i] != '.' {
			ret = append(ret, buff[i])

		} else {
			ret[index] = byte(len(ret) - l)
			ret = append(ret, 0)
			index = len(ret) - 1
			l = len(ret)
		}
	}
	return ret
}
