package src

import (
  b64 "encoding/base64"
  "fmt"
  "strings"
)

func uploadLogo(input, filename string) (logoURL string, err error) {

  b64data := input[strings.IndexByte(input, ',')+1:]

  // BAse64 in bytes umwandeln un speichern
  sDec, err := b64.StdEncoding.DecodeString(b64data)
  if err != nil {
    return
  }

  var file = fmt.Sprintf("%s%s", System.Folder.ImagesUpload, filename)

  err = writeByteToFile(file, sDec)
  if err != nil {
    return
  }

  // Respect Force HTTPS setting when generating logo URL
  if Settings.ForceHttps && Settings.HttpsThreadfinDomain != "" {
    logoURL = fmt.Sprintf("https://%s:%d/data_images/%s", Settings.HttpsThreadfinDomain, Settings.HttpsPort, filename)
  } else if Settings.HttpThreadfinDomain != "" {
    logoURL = fmt.Sprintf("http://%s:%s/data_images/%s", Settings.HttpThreadfinDomain, Settings.Port, filename)
  } else {
  logoURL = fmt.Sprintf("%s://%s/data_images/%s", System.ServerProtocol.XML, System.Domain, filename)
  }

  return

}
