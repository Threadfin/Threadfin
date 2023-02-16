
class MainMenu {
  DocumentID: string = "main-menu"
  HTMLTag: string = "LI"
  ImagePath: string = "img/"

  createIMG(src): any {
    var element = document.createElement("IMG")
    element.setAttribute("src", this.ImagePath + src)
    return element
  }

  createValue(value): any {
    var element = document.createElement("P")
    element.innerHTML = value
    return element
  }
}

class MainMenuItem extends MainMenu {
  menuKey: string
  value: string
  imgSrc: string
  headline: string
  id: string
  tableHeader: string[]

  constructor(menuKey: string, value: string, image: string, headline: string) {
    super()
    this.menuKey = menuKey
    this.value = value
    this.imgSrc = image
    this.headline = headline
  }

  createItem(): void {
    var item = document.createElement("LI")
    item.setAttribute("onclick", "javascript: openThisMenu(this)")
    item.setAttribute("id", this.id)
    item.setAttribute("class", "nav-item")
    var img = this.createIMG(this.imgSrc)
    var value = this.createValue(this.value)

    item.appendChild(img)
    item.appendChild(value)

    var doc = document.getElementById(this.DocumentID)
    doc.appendChild(item)

    switch (this.menuKey) {
      case "playlist":
        this.tableHeader = ["{{.playlist.table.playlist}}", "{{.playlist.table.tuner}}", "{{.playlist.table.lastUpdate}}", "{{.playlist.table.availability}} %", "{{.playlist.table.type}}", "{{.playlist.table.streams}}", "{{.playlist.table.groupTitle}} %", "{{.playlist.table.tvgID}} %", "{{.playlist.table.uniqueID}} %"]
        break

      case "xmltv":
        this.tableHeader = ["{{.xmltv.table.guide}}", "{{.xmltv.table.lastUpdate}}", "{{.xmltv.table.availability}} %", "{{.xmltv.table.channels}}", "{{.xmltv.table.programs}}"]
        break

      case "filter":
        this.tableHeader = ["{{.filter.table.name}}", "{{.filter.table.type}}", "{{.filter.table.filter}}"]
        break

      case "users":
        this.tableHeader = ["{{.users.table.username}}", "{{.users.table.password}}", "{{.users.table.web}}", "{{.users.table.pms}}", "{{.users.table.m3u}}", "{{.users.table.xml}}", "{{.users.table.api}}"]
        break

      case "mapping":
        this.tableHeader = ["BULK", "{{.mapping.table.chNo}}", "{{.mapping.table.logo}}", "{{.mapping.table.channelName}}", "{{.mapping.table.playlist}}", "{{.mapping.table.groupTitle}}", "{{.mapping.table.xmltvFile}}", "{{.mapping.table.xmltvID}}"]
        break

    }

    //console.log(this.menuKey, this.tableHeader);

  }
}

class Content {

  DocumentID: string = "content"
  HeaderID: string = "popup_header"
  FooterID: string = "popup_footer"
  TableID: string = "content_table"
  InactiveTableID: string = "inactive_content_table"
  DivID: string
  headerClass: string = "content_table_header"
  headerClassInactive: string = "inactive_content_table_header"
  interactionID: string = "content-interaction"

  createHeadline(value): any {
    var element = document.createElement("H3")
    element.innerHTML = value
    return element
  }

  createHR(): any {
    var element = document.createElement("HR")
    return element
  }

  createBR(): any {
    var element = document.createElement("BR")
    return element
  }

  createInteraction(): any {
    var element = document.createElement("DIV")
    element.setAttribute("id", this.interactionID)
    return element
  }

  createDIV(): any {
    var element = document.createElement("DIV")
    element.id = this.DivID
    return element
  }

  createTABLE(): any {
    var element = document.createElement("TABLE")
    element.setAttribute('class', 'table')
    element.id = this.TableID
    return element
  }

  createTableRow(): any {
    var element = document.createElement("TR")
    element.className = this.headerClass
    return element
  }

  createInactiveTABLE(): any {
    var element = document.createElement("TABLE")
    element.id = this.InactiveTableID
    return element
  }

  createInactiveTableRow(): any {
    var element = document.createElement("TR")
    element.className = this.headerClassInactive
    return element
  }

  createTableContent(menuKey: string): string[] {

    var data = new Object()
    var rows = new Array()

    switch (menuKey) {
      case "playlist":
        var fileTypes = new Array("m3u", "hdhr")

        fileTypes.forEach(fileType => {

          data = SERVER["settings"]["files"][fileType]

          var keys = getObjKeys(data)

          keys.forEach(key => {
            var tr = document.createElement("TR")
            tr.id = key

            tr.setAttribute('onclick', 'javascript: openPopUp("' + fileType + '", this)')

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["name"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            if (SERVER["settings"]["buffer"] != "-") {
              cell.value = data[key]["tuner"]
            } else {
              cell.value = "-"
            }

            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["last.update"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["provider.availability"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["type"].toUpperCase();
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["compatibility"]["streams"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["compatibility"]["group.title"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["compatibility"]["tvg.id"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["compatibility"]["stream.id"]
            tr.appendChild(cell.createCell())

            rows.push(tr)
          });

        });
        break

      case "filter":
        delete SERVER["settings"]["filter"][-1]
        data = SERVER["settings"]["filter"]
        var keys = getObjKeys(data)
        keys.forEach(key => {
          var tr = document.createElement("TR")
          tr.id = key

          tr.setAttribute('onclick', 'javascript: openPopUp("' + data[key]["type"] + '", this)')

          var cell: Cell = new Cell()
          cell.child = true
          cell.childType = "P"
          cell.value = data[key]["name"]
          tr.appendChild(cell.createCell())

          var cell: Cell = new Cell()
          cell.child = true
          cell.childType = "P"
          switch (data[key]["type"]) {
            case "custom-filter":
              cell.value = "{{.filter.custom}}"
              break;

            case "group-title":
              cell.value = "{{.filter.group}}"
              break;

            default:
              break;
          }

          tr.appendChild(cell.createCell())

          var cell: Cell = new Cell()
          cell.child = true
          cell.childType = "P"
          cell.value = data[key]["filter"]
          tr.appendChild(cell.createCell())

          rows.push(tr)

        });
        break

      case "xmltv":
        var fileTypes = new Array("xmltv")

        fileTypes.forEach(fileType => {

          data = SERVER["settings"]["files"][fileType]

          var keys = getObjKeys(data)

          keys.forEach(key => {
            var tr = document.createElement("TR")

            tr.id = key
            tr.setAttribute('onclick', 'javascript: openPopUp("' + fileType + '", this)')

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["name"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["last.update"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["provider.availability"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["compatibility"]["xmltv.channels"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["compatibility"]["xmltv.programs"]
            tr.appendChild(cell.createCell())

            rows.push(tr)
          });

        });
        break

      case "users":
        var fileTypes = new Array("users")

        fileTypes.forEach(fileType => {
          data = SERVER[fileType]

          var keys = getObjKeys(data)

          keys.forEach(key => {
            var tr = document.createElement("TR")
            tr.id = key
            tr.setAttribute('onclick', 'javascript: openPopUp("' + fileType + '", this)')

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["data"]["username"]
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = "******"
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            if (data[key]["data"]["authentication.web"] == true) {
              cell.value = "✓"
            } else {
              cell.value = "-"
            }
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            if (data[key]["data"]["authentication.pms"] == true) {
              cell.value = "✓"
            } else {
              cell.value = "-"
            }
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            if (data[key]["data"]["authentication.m3u"] == true) {
              cell.value = "✓"
            } else {
              cell.value = "-"
            }
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            if (data[key]["data"]["authentication.xml"] == true) {
              cell.value = "✓"
            } else {
              cell.value = "-"
            }
            tr.appendChild(cell.createCell())

            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            if (data[key]["data"]["authentication.api"] == true) {
              cell.value = "✓"
            } else {
              cell.value = "-"
            }
            tr.appendChild(cell.createCell())

            rows.push(tr)
          });

        });
        break

      case "mapping":
        BULK_EDIT = false
        createSearchObj()
        checkUndo("epgMapping")
        console.log("MAPPING")
        data = SERVER["xepg"]["epgMapping"]

        var keys = getObjKeys(data)
        keys.forEach(key => {
          if (data[key]["x-active"]) {
            var tr = document.createElement("TR")
            tr.id = key
            tr.className = "activeEPG"

            // Bulk
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "BULK"
            cell.value = false
            tr.appendChild(cell.createCell())

            // Kanalnummer
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "INPUTCHANNEL"
            cell.value = data[key]["x-channelID"]
            //td.setAttribute('onclick', 'javascript: changeChannelNumber("' + key + '", this)')
            tr.appendChild(cell.createCell())

            // Logo
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "IMG"
            cell.imageURL = data[key]["tvg-logo"]
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key

            tr.appendChild(td)

            // Kanalname
            var cell: Cell = new Cell()
            var cats = data[key]["x-category"].split(":")
            cell.child = true
            cell.childType = "P"
            cell.className = "category"
            var catColorSettings = SERVER["settings"]["epgCategoriesColors"]
            var colors_split = catColorSettings.split("|")
            for (var i=0; i < colors_split.length; i++) {
              var catsColor_split = colors_split[i].split(":")
              if (catsColor_split[0] == cats[0]) {
                cell.classColor = catsColor_split[1]
              }
            }
            cell.value = data[key]["x-name"]
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key
            tr.appendChild(td)


            // Playlist
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            //cell.value = data[key]["_file.m3u.name"] 
            cell.value = getValueFromProviderFile(data[key]["_file.m3u.id"], "m3u", "name")
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key
            tr.appendChild(td)


            // Gruppe (group-title)
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["x-group-title"]
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key
            tr.appendChild(td)

            // XMLTV Datei
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"

            if (data[key]["x-xmltv-file"] != "-") {
              cell.value = getValueFromProviderFile(data[key]["x-xmltv-file"], "xmltv", "name")
            } else {
              cell.value = data[key]["x-xmltv-file"]
            }

            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key
            tr.appendChild(td)

            // XMLTV Kanal
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            //var value = str.substring(1, 4);
            var value = data[key]["x-mapping"]
            if (value.length > 20) {
              value = data[key]["x-mapping"].substring(0, 20) + "..."
            }
            cell.value = value
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key

            tr.appendChild(td)

            rows.push(tr)
          }
        });

        break

      case "settings":
        alert()
        break

      default:
        console.log("Table content (menuKey):", menuKey);

        break

    }

    return rows

  }

  createInactiveTableContent(menuKey: string): string[] {

    var data = new Object()
    var rows = new Array()

    switch (menuKey) {
      case "mapping":
        BULK_EDIT = false
        createSearchObj()
        checkUndo("epgMapping")
        console.log("MAPPING")
        data = SERVER["xepg"]["epgMapping"]

        var keys = getObjKeys(data)
        keys.forEach(key => {
          if (data[key]["x-active"] === false) {

            var tr = document.createElement("TR")
            tr.id = key
            tr.className = "notActiveEPG"

            // Bulk
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "BULK"
            cell.value = false
            tr.appendChild(cell.createCell())

            // Kanalnummer
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "INPUTCHANNEL"
            if (data[key]["x-active"] == true) {
              cell.value = data[key]["x-channelID"]
            } else {
              cell.value = data[key]["x-channelID"] * 10
            }
            //td.setAttribute('onclick', 'javascript: changeChannelNumber("' + key + '", this)')
            tr.appendChild(cell.createCell())

            // Logo
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "IMG"
            cell.imageURL = data[key]["tvg-logo"]
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key

            tr.appendChild(td)

            // Kanalname
            var cell: Cell = new Cell()
            var cats = data[key]["x-category"].split(":")
            cell.child = true
            cell.childType = "P"
            cell.className = "category"
            var catColorSettings = SERVER["settings"]["epgCategoriesColors"]
            var colors_split = catColorSettings.split("|")
            for (var i=0; i < colors_split.length; i++) {
              var catsColor_split = colors_split[i].split(":")
              if (catsColor_split[0] == cats[0]) {
                cell.classColor = catsColor_split[1]
              }
            }
            cell.value = data[key]["x-name"]
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key
            tr.appendChild(td)


            // Playlist
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            //cell.value = data[key]["_file.m3u.name"] 
            cell.value = getValueFromProviderFile(data[key]["_file.m3u.id"], "m3u", "name")
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key
            tr.appendChild(td)


            // Gruppe (group-title)
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            cell.value = data[key]["x-group-title"]
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key
            tr.appendChild(td)

            // XMLTV Datei
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"

            if (data[key]["x-xmltv-file"] != "-") {
              cell.value = getValueFromProviderFile(data[key]["x-xmltv-file"], "xmltv", "name")
            } else {
              cell.value = data[key]["x-xmltv-file"]
            }

            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key
            tr.appendChild(td)

            // XMLTV Kanal
            var cell: Cell = new Cell()
            cell.child = true
            cell.childType = "P"
            //var value = str.substring(1, 4);
            var value = data[key]["x-mapping"]
            if (value.length > 20) {
              value = data[key]["x-mapping"].substring(0, 20) + "..."
            }
            cell.value = value
            var td = cell.createCell()
            td.setAttribute('onclick', 'javascript: openPopUp("mapping", this)')
            td.id = key

            tr.appendChild(td)

            rows.push(tr)
          }
        });

        break

      case "settings":
        alert()
        break

      default:
        console.log("Table content (menuKey):", menuKey);

        break

    }

    return rows

  }

  return
}

class Cell {
  child: Boolean
  childType: string
  active: Boolean
  value: any
  className: string
  classColor: string
  tdClassName: string
  imageURL: string
  onclick: boolean
  onclickFunktion: string

  createCell(): any {
    var td = document.createElement("TD")


    if (this.child == true) {
      var element: any

      switch (this.childType) {
        case "P":
          element = document.createElement(this.childType);
          element.innerHTML = this.value
          element.className = this.className
          if (this.classColor) {
            element.style.borderColor = this.classColor
          }
          break

        case "INPUT":
          element = document.createElement(this.childType);
          (element as HTMLInputElement).value = this.value;
          (element as HTMLInputElement).type = "text";
          break

        case "INPUTCHANNEL":
          element = document.createElement("INPUT");
          (element as HTMLInputElement).setAttribute("onchange", "javscript: changeChannelNumber(this)");
          (element as HTMLInputElement).value = this.value;
          (element as HTMLInputElement).type = "text";
          break

        case "BULK":
          element = document.createElement("INPUT");
          (element as HTMLInputElement).checked = this.value;
          (element as HTMLInputElement).type = "checkbox";
          (element as HTMLInputElement).className = "bulk hideBulk";
          break

        case "BULK_HEAD":
          element = document.createElement("INPUT");
          (element as HTMLInputElement).checked = this.value;
          (element as HTMLInputElement).type = "checkbox";
          (element as HTMLInputElement).className = "bulk hideBulk";
          if (this.active) {
            (element as HTMLInputElement).setAttribute("onclick", "javascript: selectAllChannels()")
          } else {
            (element as HTMLInputElement).setAttribute("onclick", "javascript: selectAllChannels('inactive_content_table')")
          }
          break

        case "IMG":
          element = document.createElement(this.childType);
          element.setAttribute("src", this.imageURL)
          if (this.imageURL != "") {
            element.setAttribute("onerror", "javascript: this.onerror=null;this.src=''")
            //onerror="this.onerror=null;this.src='missing.gif';"
          }
      }

      td.appendChild(element)

    } else {
      td.innerHTML = this.value
    }

    if (this.onclick == true) {
      td.setAttribute("onclick", this.onclickFunktion)
      td.className = "pointer"
    }

    if (this.tdClassName != undefined) {
      td.className = this.tdClassName
    }

    return td
  }

  return
}

class ShowContent extends Content {
  menuID: number

  constructor(menuID: number) {
    super()
    this.menuID = menuID
  }

  createInput(type: string, name: string, value: string,): any {

    var input = document.createElement("INPUT")
    input.setAttribute("type", type)
    input.setAttribute("name", name)
    input.setAttribute("value", value)
    return input
  }

  show(): void {
    COLUMN_TO_SORT = -1
    // Alten Inhalt löschen
    var doc = document.getElementById(this.DocumentID)
    doc.innerHTML = ""
    showPreview(false)

    // Überschrift
    var popup_header = document.getElementById(this.HeaderID)
    var headline: string[] = menuItems[this.menuID].headline

    var menuKey = menuItems[this.menuID].menuKey
    var h = this.createHeadline(headline)
    var existingHeader = popup_header.querySelector('h3')
    if(existingHeader) {
      popup_header.replaceChild(h, existingHeader)
    } else {
      popup_header.appendChild(h)
    }

    var hr = this.createHR()
    doc.appendChild(hr)

    // Interaktion
    var div = this.createInteraction()
    doc.appendChild(div)
    var interaction = document.getElementById(this.interactionID)
    switch (menuKey) {
      case "playlist":
        var input = this.createInput("button", menuKey, "{{.button.new}}")
        input.setAttribute("id", "-")
        input.setAttribute("onclick", 'javascript: openPopUp("playlist")')
        input.setAttribute('data-bs-toggle', 'modal')
        input.setAttribute('data-bs-target', '#popup')
        interaction.appendChild(input)
        break;

      case "filter":
        var input = this.createInput("button", menuKey, "{{.button.new}}")
        input.setAttribute("id", -1)
        input.setAttribute("onclick", 'javascript: openPopUp("filter", this)')
        input.setAttribute('data-bs-toggle', 'modal')
        input.setAttribute('data-bs-target', '#popup')
        interaction.appendChild(input)
        break;


      case "xmltv":
        var input = this.createInput("button", menuKey, "{{.button.new}}")
        input.setAttribute("id", "xmltv")
        input.setAttribute("onclick", 'javascript: openPopUp("xmltv")')
        input.setAttribute('data-bs-toggle', 'modal')
        input.setAttribute('data-bs-target', '#popup')
        interaction.appendChild(input)
        break;

      case "users":
        var input = this.createInput("button", menuKey, "{{.button.new}}")
        input.setAttribute("id", "users")
        input.setAttribute("onclick", 'javascript: openPopUp("users")')
        input.setAttribute('data-bs-toggle', 'modal')
        input.setAttribute('data-bs-target', '#popup')
        interaction.appendChild(input)
        break;

      case "mapping":
        // showElement("loading", true)
        var input = this.createInput("button", menuKey, "{{.button.save}}")
        input.setAttribute("onclick", 'javascript: savePopupData("mapping", "", "")')
        interaction.appendChild(input)

        var input = this.createInput("button", menuKey, "{{.button.bulkEdit}}")
        input.setAttribute("onclick", 'javascript: bulkEdit()')
        interaction.appendChild(input)

        var input = this.createInput("search", "search", "")
        input.setAttribute("id", "searchMapping")
        input.setAttribute("placeholder", "{{.button.search}}")
        input.className = "search"
        input.setAttribute("onchange", 'javascript: searchInMapping()')
        interaction.appendChild(input)
        break;

      case "settings":
        var input = this.createInput("button", menuKey, "{{.button.save}}")
        input.setAttribute("onclick", 'javascript: saveSettings();')
        interaction.appendChild(input)

        var input = this.createInput("button", menuKey, "{{.button.backup}}")
        input.setAttribute("onclick", 'javascript: backup();')
        interaction.appendChild(input)

        var input = this.createInput("button", menuKey, "{{.button.restore}}")
        input.setAttribute("onclick", 'javascript: restore();')
        interaction.appendChild(input)

        var wrapper = document.createElement("DIV")
        wrapper.setAttribute("id", "box-wrapper")
        doc.appendChild(wrapper)

        this.DivID = "content_settings"
        var settings = this.createDIV()
        wrapper.appendChild(settings)

        showSettings()

        return
        break

      case "log":
        var input = this.createInput("button", menuKey, "{{.button.resetLogs}}")
        input.setAttribute("onclick", 'javascript: resetLogs();')
        interaction.appendChild(input)

        var wrapper = document.createElement("DIV")
        wrapper.setAttribute("id", "box-wrapper")
        doc.appendChild(wrapper)

        this.DivID = "content_log"
        var logs = this.createDIV()
        wrapper.appendChild(logs)

        showLogs(true)

        return
        break

      case "logout":
        location.reload()
        document.cookie = "Token= ; expires = Thu, 01 Jan 1970 00:00:00 GMT"
        break

      default:
        console.log("Show content (menuKey):", menuKey);
        break;
    }

    // Tabelle erstellen (falls benötigt)
    var tableHeader: string[] = menuItems[this.menuID].tableHeader
    if (tableHeader.length > 0) {
      var wrapper = document.createElement("DIV")
      doc.appendChild(wrapper)
      wrapper.setAttribute("id", "box-wrapper")

      var table = this.createTABLE()
      wrapper.appendChild(table)

      var header = this.createTableRow()
      table.appendChild(header)

      // Kopfzeile der Tablle
      tableHeader.forEach(element => {
        var cell: Cell = new Cell()
        cell.child = true
        cell.childType = "P"
        cell.value = element
        if (element == "BULK") {
          cell.childType = "BULK_HEAD";
          cell.active = true
          cell.value = false
        }

        if (menuKey == "mapping") {

          if (element == "{{.mapping.table.chNo}}") {
            cell.onclick = true
            cell.onclickFunktion = "javascript: sortTable(1);"
            cell.tdClassName = "sortThis"
          }

          if (element == "{{.mapping.table.channelName}}") {
            cell.onclick = true
            cell.onclickFunktion = "javascript: sortTable(3);"
          }

          if (element == "{{.mapping.table.playlist}}") {
            cell.onclick = true
            cell.onclickFunktion = "javascript: sortTable(4);"
          }

          if (element == "{{.mapping.table.groupTitle}}") {
            cell.onclick = true
            cell.onclickFunktion = "javascript: sortTable(5);"
          }

        }

        header.appendChild(cell.createCell())
      });

      table.appendChild(header)

      // Inhalt der Tabelle
      var rows: any = this.createTableContent(menuKey)
      rows.forEach(tr => {
        table.appendChild(tr)
      });

      var br = this.createBR()
      doc.appendChild(br)

      // Create inactive channels for mapping
      if (menuKey == "mapping") {


        var inactivetable = this.createInactiveTABLE()
        wrapper.appendChild(inactivetable)

        var header = this.createInactiveTableRow()
        inactivetable.appendChild(header)

        // Kopfzeile der Tablle
        tableHeader.forEach(element => {
          var cell: Cell = new Cell()
          cell.child = true
          cell.childType = "P"
          cell.value = element
          if (element == "BULK") {
            cell.childType = "BULK_HEAD";
            cell.active = false
            cell.value = false
          }

          if (menuKey == "mapping") {

            if (element == "{{.mapping.table.chNo}}") {
              cell.onclick = true
              cell.onclickFunktion = "javascript: sortTable(1, 'inactive_content_table');"
              cell.tdClassName = "sortThis"
            }

            if (element == "{{.mapping.table.channelName}}") {
              cell.onclick = true
              cell.onclickFunktion = "javascript: sortTable(3, 'inactive_content_table');"
            }

            if (element == "{{.mapping.table.playlist}}") {
              cell.onclick = true
              cell.onclickFunktion = "javascript: sortTable(4, 'inactive_content_table');"
            }

            if (element == "{{.mapping.table.groupTitle}}") {
              cell.onclick = true
              cell.onclickFunktion = "javascript: sortTable(5, 'inactive_content_table');"
            }

          }

          header.appendChild(cell.createCell())
        });

        inactivetable.appendChild(header)

        // Inhalt der Tabelle
        var rows: any = this.createInactiveTableContent(menuKey)
        rows.forEach(tr => {
          inactivetable.appendChild(tr)
        });
        savePopupData("mapping", "", false, 0)

      }
    }

    switch (menuKey) {
      case "mapping":
        sortTable(1)
        sortTable(1, "inactive_content_table")
        break;

      case "filter":
        showPreview(true)
        sortTable(0)
        break

      default:
        COLUMN_TO_SORT = -1
        sortTable(0)
        break;
    }

    showElement("loading", false)
  }

}

function PageReady() {

  var server: Server = new Server("getServerConfig")
  server.request(new Object())

  setInterval(function () {
    updateLog()
  }, 10000);


  return
}

function createLayout() {

  // Client Info
  var obj = SERVER["clientInfo"]
  var keys = getObjKeys(obj);
  for (var i = 0; i < keys.length; i++) {

    if (document.getElementById(keys[i])) {
      (<HTMLInputElement>document.getElementById(keys[i])).value = obj[keys[i]];
    }

  }

  if (!document.getElementById("main-menu")) {
    return
  }



  // Menü erstellen
  document.getElementById("main-menu").innerHTML = ""
  for (let i = 0; i < menuItems.length; i++) {

    menuItems[i].id = i

    switch (menuItems[i]["menuKey"]) {

      case "users":
      case "logout":
        if (SERVER["settings"]["authentication.web"] == true) {
          menuItems[i].createItem()
        }
        break

      case "mapping":
      case "xmltv":
        if (SERVER["clientInfo"]["epgSource"] == "XEPG") {
          menuItems[i].createItem()
        }
        break

      default:
        menuItems[i].createItem()
        break
    }

  }

  return
}

function openThisMenu(element) {
  var id = element.id
  var content: ShowContent = new ShowContent(id)
  content.show()
  enableGroupSelection(".bulk")
  return
}

class PopupWindow {
  DocumentID: string = "popup-custom"
  InteractionID: string = "interaction"
  doc = document.getElementById(this.DocumentID)

  createTitle(title: string): any {
    var td = document.createElement("TD")
    td.className = "left"
    td.innerHTML = title + ":"
    return td
  }

  createContent(element): any {
    var td = document.createElement("TD")
    td.appendChild(element)
    return td
  }

  createInteraction(): any {
    var div = document.createElement("div")
    div.setAttribute("id", "popup-interaction")
    div.className = "interaction"
    this.doc.appendChild(div)
  }
}

class PopupContent extends PopupWindow {

  table = document.createElement("TABLE")

  createHeadline(headline): void {
    this.doc.innerHTML = ""
    var element = document.createElement("H3")
    element.innerHTML = headline.toUpperCase()
    this.doc.appendChild(element)

    // Tabelle erstellen
    this.table = document.createElement("TABLE")
    this.doc.appendChild(this.table)
  }

  appendRow(title: string, element: any): void {
    var tr = document.createElement("TR")

    // Bezeichnung
    if (title.length != 0) {
      tr.appendChild(this.createTitle(title))
    }


    // Content
    tr.appendChild(this.createContent(element))
    this.table.appendChild(tr)
  }


  createInput(type: string, name: string, value: string): any {

    var input = document.createElement("INPUT")
    if (value == undefined) {
      value = ""
    }

    input.setAttribute("type", type)
    input.setAttribute("name", name)
    input.setAttribute("value", value)
    return input
  }

  createCheckbox(name: string): any {
    var input = document.createElement("INPUT")

    input.setAttribute("type", "checkbox")
    input.setAttribute("name", name)
    return input
  }

  createSelect(text: string[], values: string[], set: string, dbKey: string): any {
    var select = document.createElement("SELECT")
    select.setAttribute("name", dbKey)
    for (let i = 0; i < text.length; i++) {
      var option = document.createElement("OPTION")
      option.setAttribute("value", values[i])
      option.innerText = text[i]
      select.appendChild(option)
    }
    if (set != "") {
      (select as HTMLSelectElement).value = set
    }

    if (set == undefined) {
      (select as HTMLSelectElement).value = values[0]
    }

    return select
  }

  selectOption(select: any, value: string): any {
    //select.selectedOptions = value
    var s: HTMLSelectElement = (select as HTMLSelectElement)
    s.options[s.selectedIndex].value = value
    return select
  }

  description(value: string): any {
    var tr = document.createElement("TR")
    var td = document.createElement("TD")
    var span = document.createElement("PRE")

    span.innerHTML = value

    tr.appendChild(td)

    tr.appendChild(this.createContent(span))

    this.table.appendChild(tr)
  }

  // Interaktion
  addInteraction(element: any) {
    var interaction = document.getElementById("popup-interaction")
    interaction.appendChild(element)
  }
}

function openPopUp(dataType, element) {

  var data: object = new Object();
  var id: any
  switch (element) {
    case undefined:

      switch (dataType) {
        case "group-title":
          if (id == undefined) {
            id = -1
          }
          data = getLocalData("filter", id)
          data["type"] = "group-title"
          break;

        case "custom-filter":
          if (id == undefined) {
            id = -1
          }
          data = getLocalData("filter", id)
          data["type"] = "custom-filter"
          break;

        default:
          data["id.provider"] = "-"
          data["type"] = dataType
          id = "-"
          break;
      }

      break

    default:
      id = element.id
      data = getLocalData(dataType, id)
      break;
  }

  var content: PopupContent = new PopupContent()

  switch (dataType) {
    case "playlist":
      content.createHeadline("{{.playlist.playlistType.title}}")
      // Type
      var text: string[] = ["M3U", "HDHomeRun"]
      var values: string[] = ["javascript: openPopUp('m3u')", "javascript: openPopUp('hdhr')"]
      var select = content.createSelect(text, values, "", "type")
      select.setAttribute("id", "type")
      select.setAttribute("onchange", 'javascript: changeButtonAction(this, "next", "onclick")') // changeButtonAction
      content.appendRow("{{.playlist.type.title}}", select)

      // Interaktion
      content.createInteraction()
      // Abbrechen
      var input = content.createInput("button", "cancel", "{{.button.cancel}}")
      input.setAttribute("onclick", 'javascript: showElement("popup", false);')
      content.addInteraction(input)

      // Weiter
      var input = content.createInput("button", "next", "{{.button.next}}")
      input.setAttribute("onclick", 'javascript: openPopUp("m3u")')
      input.setAttribute("id", 'next')
      content.addInteraction(input)
      break

    case "m3u":
      content.createHeadline(dataType)
      // Name
      var dbKey: string = "name"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.playlist.name.placeholder}}")
      content.appendRow("{{.playlist.name.title}}", input)

      // Beschreibung
      var dbKey: string = "description"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.playlist.description.placeholder}}")
      content.appendRow("{{.playlist.description.title}}", input)

      // URL
      var dbKey: string = "file.source"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.playlist.fileM3U.placeholder}}")
      content.appendRow("{{.playlist.fileM3U.title}}", input)

      // Tuner
      if (SERVER["settings"]["buffer"] != "-") {
        var text: string[] = new Array()
        var values: string[] = new Array()

        for (var i = 1; i <= 100; i++) {
          text.push(i.toString())
          values.push(i.toString())
        }

        var dbKey: string = "tuner"
        var select = content.createSelect(text, values, data[dbKey], dbKey)
        select.setAttribute("onfocus", "javascript: return;")
        content.appendRow("{{.playlist.tuner.title}}", select)
      } else {
        var dbKey: string = "tuner"
        if (data[dbKey] == undefined) {
          data[dbKey] = 1
        }
        var input = content.createInput("text", dbKey, data[dbKey])
        input.setAttribute("readonly", "true")
        input.className = "notAvailable"
        content.appendRow("{{.playlist.tuner.title}}", input)
      }

      content.description("{{.playlist.tuner.description}}")

      // Interaktion
      content.createInteraction()
      // Löschen
      if (data["id.provider"] != "-") {
        var input = content.createInput("button", "delete", "{{.button.delete}}")
        input.className = "delete"
        input.setAttribute('onclick', 'javascript: savePopupData("m3u", "' + id + '", true, 0)')
        content.addInteraction(input)
      } else {
        var input = content.createInput("button", "back", "{{.button.back}}")
        input.setAttribute("onclick", 'javascript: openPopUp("playlist")')
        content.addInteraction(input)
      }

      // Abbrechen
      var input = content.createInput("button", "cancel", "{{.button.cancel}}")
      input.setAttribute("onclick", 'javascript: showElement("popup", false);')
      content.addInteraction(input)

      // Aktualisieren
      if (data["id.provider"] != "-") {
        var input = content.createInput("button", "update", "{{.button.update}}")
        input.setAttribute('onclick', 'javascript: savePopupData("m3u", "' + id + '", false, 1)')
        content.addInteraction(input)
      }

      // Speichern
      var input = content.createInput("button", "save", "{{.button.save}}")
      input.setAttribute('onclick', 'javascript: savePopupData("m3u", "' + id + '", false, 0)')
      content.addInteraction(input)
      break

    case "hdhr":
      content.createHeadline(dataType)
      // Name
      var dbKey: string = "name"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.playlist.name.placeholder}}")
      content.appendRow("{{.playlist.name.title}}", input)

      // Beschreibung
      var dbKey: string = "description"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.playlist.description.placeholder}}")
      content.appendRow("{{.playlist.description.placeholder}}", input)

      // URL
      var dbKey: string = "file.source"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.playlist.fileHDHR.placeholder}}")
      content.appendRow("{{.playlist.fileHDHR.title}}", input)

      // Tuner
      if (SERVER["settings"]["buffer"] != "-") {
        var text: string[] = new Array()
        var values: string[] = new Array()

        for (var i = 1; i <= 100; i++) {
          text.push(i.toString())
          values.push(i.toString())
        }

        var dbKey: string = "tuner"
        var select = content.createSelect(text, values, data[dbKey], dbKey)
        select.setAttribute("onfocus", "javascript: return;")
        content.appendRow("{{.playlist.tuner.title}}", select)
      } else {
        var dbKey: string = "tuner"
        if (data[dbKey] == undefined) {
          data[dbKey] = 1
        }
        var input = content.createInput("text", dbKey, data[dbKey])
        input.setAttribute("readonly", "true")
        input.className = "notAvailable"
        content.appendRow("{{.playlist.tuner.title}}", input)
      }

      content.description("{{.playlist.tuner.description}}")

      // Interaktion
      content.createInteraction()
      // Löschen
      if (data["id.provider"] != "-") {
        var input = content.createInput("button", "delete", "{{.button.delete}}")
        input.setAttribute('onclick', 'javascript: savePopupData("hdhr", "' + id + '", true, 0)')
        input.className = "delete"
        content.addInteraction(input)
      } else {
        var input = content.createInput("button", "back", "{{.button.back}}")
        input.setAttribute("onclick", 'javascript: openPopUp("playlist")')
        content.addInteraction(input)
      }

      // Abbrechen
      var input = content.createInput("button", "cancel", "{{.button.cancel}}")
      input.setAttribute("onclick", 'javascript: showElement("popup", false);')
      content.addInteraction(input)

      // Aktualisieren
      if (data["id.provider"] != "-") {
        var input = content.createInput("button", "update", "{{.button.update}}")
        input.setAttribute('onclick', 'javascript: savePopupData("hdhr", "' + id + '", false, 1)')
        content.addInteraction(input)
      }

      // Speichern
      var input = content.createInput("button", "save", "{{.button.save}}")
      input.setAttribute('onclick', 'javascript: savePopupData("hdhr", "' + id + '", false, 0)')
      content.addInteraction(input)
      break

    case "filter":
      content.createHeadline(dataType)

      // Type
      var dbKey: string = "type"
      var text: string[] = ["M3U: " + "{{.filter.type.groupTitle}}", "Threadfin: " + "{{.filter.type.customFilter}}"]
      var values: string[] = ["javascript: openPopUp('group-title')", "javascript: openPopUp('custom-filter')"]
      var select = content.createSelect(text, values, "javascript: openPopUp('group-title')", dbKey)
      select.setAttribute("id", id)
      select.setAttribute("onchange", 'javascript: changeButtonAction(this, "next", "onclick");') // changeButtonAction
      content.appendRow("{{.filter.type.title}}", select)

      // Interaktion
      content.createInteraction()
      // Abbrechen
      var input = content.createInput("button", "cancel", "{{.button.cancel}}")
      input.setAttribute("onclick", 'javascript: showElement("popup", false);')
      content.addInteraction(input)

      // Weiter
      var input = content.createInput("button", "next", "{{.button.next}}")
      input.setAttribute("onclick", 'javascript: openPopUp("group-title")')
      input.setAttribute("id", 'next')
      content.addInteraction(input)
      break

    case "custom-filter":
    case "group-title":

      switch (dataType) {
        case "custom-filter":
          content.createHeadline("{{.filter.custom}}")
          break;

        case "group-title":
          content.createHeadline("{{.filter.group}}")
          break;
      }

      // Name      
      var dbKey: string = "name"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.filter.name.placeholder}}")
      content.appendRow("{{.filter.name.title}}", input)

      // Beschreibung
      var dbKey: string = "description"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.filter.description.placeholder}}")
      content.appendRow("{{.filter.description.title}}", input)

      // Typ
      var dbKey: string = "type"
      var input = content.createInput("hidden", dbKey, data[dbKey])
      content.appendRow("", input)

      var filterType = data[dbKey]

      switch (filterType) {

        case "custom-filter":
          // Groß- Kleinschreibung beachten
          var dbKey: string = "caseSensitive"
          var input = content.createCheckbox(dbKey)
          input.checked = data[dbKey]
          content.appendRow("{{.filter.caseSensitive.title}}", input)

          // Filterregel (Benutzerdefiniert)
          var dbKey: string = "filter"
          var input = content.createInput("text", dbKey, data[dbKey])
          input.setAttribute("placeholder", "{{.filter.filterRule.placeholder}}")
          content.appendRow("{{.filter.filterRule.title}}", input)

          break;

        case "group-title":
          //alert(dbKey + " " + filterType)
          // Filter basierend auf den Gruppen in der M3U
          var dbKey: string = "filter"
          var groupsM3U = getLocalData("m3uGroups", "")
          var text: string[] = groupsM3U["text"]
          var values: string[] = groupsM3U["value"]

          var select = content.createSelect(text, values, data[dbKey], dbKey)
          select.setAttribute("onchange", "javascript: this.className = 'changed'")
          content.appendRow("{{.filter.filterGroup.title}}", select)
          content.description("{{.filter.filterGroup.description}}")

          // Groß- Kleinschreibung beachten
          var dbKey: string = "caseSensitive"
          var input = content.createCheckbox(dbKey)
          input.checked = data[dbKey]
          content.appendRow("{{.filter.caseSensitive.title}}", input)


          var dbKey: string = "include"
          var input = content.createInput("text", dbKey, data[dbKey])
          input.setAttribute("placeholder", "{{.filter.include.placeholder}}")

          content.appendRow("{{.filter.include.title}}", input)
          content.description("{{.filter.include.description}}")

          var dbKey: string = "exclude"
          var input = content.createInput("text", dbKey, data[dbKey])
          input.setAttribute("placeholder", "{{.filter.exclude.placeholder}}")
          content.appendRow("{{.filter.exclude.title}}", input)
          content.description("{{.filter.exclude.description}}")

          break

        default:
          break;
      }

      // Name      
      var dbKey: string = "startingNumber"
      if (data[dbKey] !== undefined) {
        var input = content.createInput("text", dbKey, data[dbKey])
      } else {
        var input = content.createInput("text", dbKey, "1000")
      }
      input.setAttribute("placeholder", "{{.filter.startingnumber.placeholder}}")
      content.appendRow("{{.filter.startingnumber.title}}", input)
      content.description("{{.filter.startingnumber.description}}")

      var dbKey: string = "x-category"

      var text: string[] = ["-"]
      var values: string[] = [""]
      var epgCategories = SERVER["settings"]["epgCategories"]
      var categories = epgCategories.split("|")
      
      for (i=0; i <= categories.length; i++) {
        var cat: string = categories[i]
        if (cat) {
          var cat_split: string[] = cat.split(":")
          text.push(cat_split[0])
          values.push(cat_split[1])
        }
      }

      var select = content.createSelect(text, values, data[dbKey], dbKey)
      select.setAttribute("onchange", "javascript: this.className = 'changed'")
      content.appendRow("{{.filter.category.title}}", select)

      // Interaktion
      content.createInteraction()

      // Löschen
      var input = content.createInput("button", "delete", "{{.button.delete}}")
      input.setAttribute('onclick', 'javascript: savePopupData("filter", "' + id + '", true, 0)')
      input.className = "delete"
      content.addInteraction(input)

      // Abbrechen
      var input = content.createInput("button", "cancel", "{{.button.cancel}}")
      input.setAttribute("onclick", 'javascript: showElement("popup", false);')
      content.addInteraction(input)

      // Speichern
      var input = content.createInput("button", "save", "{{.button.save}}")
      input.setAttribute('onclick', 'javascript: savePopupData("filter", "' + id + '", false, 0)')
      content.addInteraction(input)

      break

    case "xmltv":
      content.createHeadline(dataType)
      // Name
      var dbKey: string = "name"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.xmltv.name.placeholder}}")
      content.appendRow("{{.xmltv.name.title}}", input)

      // Beschreibung
      var dbKey: string = "description"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.xmltv.description.placeholder}}")
      content.appendRow("{{.xmltv.description.title}}", input)

      // URL
      var dbKey: string = "file.source"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.xmltv.fileXMLTV.placeholder}}")
      content.appendRow("{{.xmltv.fileXMLTV.title}}", input)

      // Interaktion
      content.createInteraction()
      // Löschen
      if (data["id.provider"] != "-") {
        var input = content.createInput("button", "delete", "{{.button.delete}}")
        input.setAttribute('onclick', 'javascript: savePopupData("xmltv", "' + id + '", true, 0)')
        input.className = "delete"
        content.addInteraction(input)
      }

      // Abbrechen
      var input = content.createInput("button", "cancel", "{{.button.cancel}}")
      input.setAttribute("onclick", 'javascript: showElement("popup", false);')
      content.addInteraction(input)

      // Aktualisieren
      if (data["id.provider"] != "-") {
        var input = content.createInput("button", "update", "{{.button.update}}")
        input.setAttribute('onclick', 'javascript: savePopupData("xmltv", "' + id + '", false, 1)')
        content.addInteraction(input)
      }

      // Speichern
      var input = content.createInput("button", "save", "{{.button.save}}")
      input.setAttribute('onclick', 'javascript: savePopupData("xmltv", "' + id + '", false, 0)')
      content.addInteraction(input)
      break

    case "users":
      content.createHeadline("{{.mainMenu.item.users}}")
      // Benutzername 
      var dbKey: string = "username"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.users.username.placeholder}}")
      content.appendRow("{{.users.username.title}}", input)

      // Neues Passwort 
      var dbKey: string = "password"
      var input = content.createInput("password", dbKey, "")
      input.setAttribute("placeholder", "{{.users.password.placeholder}}")
      content.appendRow("{{.users.password.title}}", input)

      // Bestätigung 
      var dbKey: string = "confirm"
      var input = content.createInput("password", dbKey, "")
      input.setAttribute("placeholder", "{{.users.confirm.placeholder}}")
      content.appendRow("{{.users.confirm.title}}", input)

      // Berechtigung WEB
      var dbKey: string = "authentication.web"
      var input = content.createCheckbox(dbKey)
      input.checked = data[dbKey]
      if (data["defaultUser"] == true) {
        input.setAttribute("onclick", "javascript: return false")
      }
      content.appendRow("{{.users.web.title}}", input)

      // Berechtigung PMS
      var dbKey: string = "authentication.pms"
      var input = content.createCheckbox(dbKey)
      input.checked = data[dbKey]
      content.appendRow("{{.users.pms.title}}", input)

      // Berechtigung M3U
      var dbKey: string = "authentication.m3u"
      var input = content.createCheckbox(dbKey)
      input.checked = data[dbKey]
      content.appendRow("{{.users.m3u.title}}", input)

      // Berechtigung XML
      var dbKey: string = "authentication.xml"
      var input = content.createCheckbox(dbKey)
      input.checked = data[dbKey]
      content.appendRow("{{.users.xml.title}}", input)

      // Berechtigung API
      var dbKey: string = "authentication.api"
      var input = content.createCheckbox(dbKey)
      input.checked = data[dbKey]
      content.appendRow("{{.users.api.title}}", input)

      // Interaktion
      content.createInteraction()

      // Löschen
      if (data["defaultUser"] != true && id != "-") {
        var input = content.createInput("button", "delete", "{{.button.delete}}")
        input.className = "delete"
        input.setAttribute('onclick', 'javascript: savePopupData("' + dataType + '", "' + id + '", true, 0)')
        content.addInteraction(input)
      }

      // Abbrechen
      var input = content.createInput("button", "cancel", "{{.button.cancel}}")
      input.setAttribute("onclick", 'javascript: showElement("popup", false);')
      content.addInteraction(input)

      // Speichern
      var input = content.createInput("button", "save", "{{.button.save}}")
      input.setAttribute("onclick", 'javascript: savePopupData("' + dataType + '", "' + id + '", "false");')
      content.addInteraction(input)

      break

    case "mapping":
      content.createHeadline("{{.mainMenu.item.mapping}}")
      if (BULK_EDIT == true) {
        var dbKey: string = "x-channels-start"
        var input = content.createInput("text", dbKey, data[dbKey])

        // Set the value to the first selected channel
        var channels = getAllSelectedChannels()
        var channel = SERVER["xepg"]["epgMapping"][channels[0]]
        if (typeof channel !== "undefined") {
          input.setAttribute("value", channel["x-channelID"])
        }

        input.setAttribute("onchange", 'javascript: changeChannelNumbers("' + channels + '");')
        content.appendRow("{{.mapping.channelGroupStart.title}}", input)
      }

      // Aktiv 
      var dbKey: string = "x-active"
      var input = content.createCheckbox(dbKey)
      input.checked = data[dbKey]
      input.id = "active"
      //input.setAttribute("onchange", "javascript: this.className = 'changed'")
      input.setAttribute("onchange", "javascript: toggleChannelStatus('" + id + "', this)")
      content.appendRow("{{.mapping.active.title}}", input)

      // Kanalname 
      var dbKey: string = "x-name"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("onchange", "javascript: this.className = 'changed'")
      if (BULK_EDIT == true) {
        input.style.border = "solid 1px red"
        input.setAttribute("readonly", "true")
      }
      content.appendRow("{{.mapping.channelName.title}}", input)

      content.description(data["name"])

      // Beschreibung 
      var dbKey: string = "x-description"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("placeholder", "{{.mapping.description.placeholder}}")
      input.setAttribute("onchange", "javascript: this.className = 'changed'")
      content.appendRow("{{.mapping.description.title}}", input)

      // Aktualisierung des Kanalnamens
      if (data.hasOwnProperty("_uuid.key")) {
        if (data["_uuid.key"] != "") {
          var dbKey: string = "x-update-channel-name"
          var input = content.createCheckbox(dbKey)
          input.setAttribute("onchange", "javascript: this.className = 'changed'")
          input.checked = data[dbKey]
          content.appendRow("{{.mapping.updateChannelName.title}}", input)
        }
      }

      // Logo URL (Kanal) 
      var dbKey: string = "tvg-logo"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("onchange", "javascript: this.className = 'changed'")
      input.setAttribute("id", "channel-icon")
      content.appendRow("{{.mapping.channelLogo.title}}", input)

      // Aktualisierung des Kanallogos
      var dbKey: string = "x-update-channel-icon"
      var input = content.createCheckbox(dbKey)
      input.checked = data[dbKey]
      input.setAttribute("id", "update-icon")
      input.setAttribute("onchange", "javascript: this.className = 'changed'; changeChannelLogo('" + id + "');")
      content.appendRow("{{.mapping.updateChannelLogo.title}}", input)

      // Erweitern der EPG Kategorie
      var dbKey: string = "x-category"
      var text: string[] = ["-"]
      var values: string[] = [""]
      var epgCategories = SERVER["settings"]["epgCategories"]
      var categories = epgCategories.split("|")
      
      for (i=0; i <= categories.length; i++) {
        var cat: string = categories[i]
        if (cat) {
          var cat_split: string[] = cat.split(":")
          text.push(cat_split[0])
          values.push(cat_split[1])
        }
      }

      var select = content.createSelect(text, values, data[dbKey], dbKey)
      select.setAttribute("onchange", "javascript: this.className = 'changed'")
      content.appendRow("{{.mapping.epgCategory.title}}", select)

      // M3U Gruppentitel
      var dbKey: string = "x-group-title"
      var input = content.createInput("text", dbKey, data[dbKey])
      input.setAttribute("onchange", "javascript: this.className = 'changed'")
      content.appendRow("{{.mapping.m3uGroupTitle.title}}", input)

      if (data["group-title"] != undefined) {
        content.description(data["group-title"])
      }

      // XMLTV Datei
      var dbKey: string = "x-xmltv-file"
      var xmlFile = data[dbKey]
      var xmltv: XMLTVFile = new XMLTVFile()
      var select = xmltv.getFiles(data[dbKey])
      select.setAttribute("name", dbKey)
      select.setAttribute("id", "popup-xmltv")
      select.setAttribute("onchange", "javascript: this.className = 'changed'; setXmltvChannel('" + id + "',this, '" + data["x-mapping"] + "');")
      content.appendRow("{{.mapping.xmltvFile.title}}", select)
      var file = data[dbKey]

      // XMLTV Mapping
      var dbKey: string = "x-mapping"
      var xmltv: XMLTVFile = new XMLTVFile()
      var select = xmltv.getPrograms(file, data[dbKey], false)
      var mappingType = data[dbKey]
      console.log("CHECKING: " + mappingType)
      select.setAttribute("name", dbKey)
      select.setAttribute("id", "popup-mapping")
      select.setAttribute("onchange", "javascript: this.className = 'changed'; checkXmltvChannel('" + id + "',this,'" + xmlFile + "');")

      sortSelect(select)
      content.appendRow("{{.mapping.xmltvChannel.title}}", select)
      
      // Extra PPV Data
      if(mappingType == "PPV") {
        var dbKey: string = "x-ppv-extra"
        var input = content.createInput("text", dbKey, data[dbKey])
        input.setAttribute("onchange", "javascript: this.className = 'changed'")
        input.setAttribute("id", "ppv-extra")
        content.appendRow("{{.mapping.ppvextra.title}}", input)
      }

      var dbKey: string = "x-backup-channel-1"
      var xmltv: XMLTVFile = new XMLTVFile()
      var select = xmltv.getPrograms(file, data[dbKey], true)
      select.setAttribute("name", dbKey)
      select.setAttribute("id", "backup-channel-1")
      select.setAttribute("onchange", "javascript: this.className = 'changed'; checkXmltvChannel('" + id + "',this,'" + xmlFile + "');")
      content.appendRow("{{.mapping.backupChannel1.title}}", select)

      var dbKey: string = "x-backup-channel-2"
      var xmltv: XMLTVFile = new XMLTVFile()
      var select = xmltv.getPrograms(file, data[dbKey], true)
      select.setAttribute("name", dbKey)
      select.setAttribute("id", "backup-channel-2")
      select.setAttribute("onchange", "javascript: this.className = 'changed'; checkXmltvChannel('" + id + "',this,'" + xmlFile + "');")
      content.appendRow("{{.mapping.backupChannel2.title}}", select)

      var dbKey: string = "x-backup-channel-3"
      var xmltv: XMLTVFile = new XMLTVFile()
      var select = xmltv.getPrograms(file, data[dbKey], true)
      select.setAttribute("name", dbKey)
      select.setAttribute("id", "backup-channel-3")
      select.setAttribute("onchange", "javascript: this.className = 'changed'; checkXmltvChannel('" + id + "',this,'" + xmlFile + "');")
      content.appendRow("{{.mapping.backupChannel3.title}}", select)

      // Interaktion
      content.createInteraction()

      // Logo hochladen
      var input = content.createInput("button", "cancel", "{{.button.uploadLogo}}")
      input.setAttribute("onclick", 'javascript: uploadLogo();')
      content.addInteraction(input)

      // Abbrechen
      var input = content.createInput("button", "cancel", "{{.button.cancel}}")
      input.setAttribute("onclick", 'javascript: showElement("popup", false);')
      content.addInteraction(input)

      // Fertig
      var ids: string[] = new Array()
      ids = getAllSelectedChannels()
      if (ids.length == 0) {
        ids.push(id)
      }

      var input = content.createInput("button", "save", "{{.button.done}}")
      input.setAttribute("onclick", 'javascript: donePopupData("' + dataType + '", "' + ids + '", "false");')
      content.addInteraction(input)
      break

    default:
      break;
  }

  showPopUpElement('popup-custom');
}

class XMLTVFile {
  File: string

  getFiles(set: string): any {
    var fileIDs: string[] = getObjKeys(SERVER["xepg"]["xmltvMap"])
    var values = new Array("-");
    var text = new Array("-");

    for (let i = 0; i < fileIDs.length; i++) {
      if (fileIDs[i] != "Threadfin Dummy") {
        values.push(getValueFromProviderFile(fileIDs[i], "xmltv", "file.threadfin"))
        text.push(getValueFromProviderFile(fileIDs[i], "xmltv", "name"))
      } else {
        values.push(fileIDs[i])
        text.push(fileIDs[i])
      }

    }

    var select = document.createElement("SELECT")
    for (let i = 0; i < text.length; i++) {
      var option = document.createElement("OPTION")
      option.setAttribute("value", values[i])
      option.innerText = text[i]
      select.appendChild(option)
    }

    if (set != "") {
      (select as HTMLSelectElement).value = set
    }

    return select
  }

  getPrograms(file: string, set: string, active: boolean): any {
    //var fileIDs:string[] = getObjKeys(SERVER["xepg"]["xmltvMap"])
    var values = getObjKeys(SERVER["xepg"]["xmltvMap"][file]);
    var text = new Array()
    var displayName: string
    var actives = getObjKeys(SERVER["data"]["StreamPreviewUI"]["activeStreams"])
    var active_list = new Array()

    if (active == true) {
      for (let i = 0; i < actives.length; i++) {        
        var names_split = SERVER["data"]["StreamPreviewUI"]["activeStreams"][actives[i]].split("[");
        displayName = names_split[0].trim();
        
        if (displayName != "") { 
          var object = {"value": displayName, "display": displayName}
          active_list.push(object)
        }
      }
    } else {
      for (let i = 0; i < values.length; i++) {
          if (SERVER["xepg"]["xmltvMap"][file][values[i]].hasOwnProperty('display-name') == true) {
            displayName = SERVER["xepg"]["xmltvMap"][file][values[i]]["display-name"];
          } else {
            displayName = "-"
          }
  
        text[i] = displayName + " (" + values[i] + ")";
      }
    }

    text.unshift("-");
    values.unshift("-");

    var select = document.createElement("SELECT")
    for (let i = 0; i < text.length; i++) {
      var option = document.createElement("OPTION")
      option.setAttribute("value", values[i])
      option.innerText = text[i]
      select.appendChild(option)
    }
    for (let i = 0; i < active_list.length; i++) {
      var option = document.createElement("OPTION")
      option.setAttribute("value", active_list[i]["value"])
      option.innerText = active_list[i]["display"]
      select.appendChild(option)
    }

    if (set != "") {
      (select as HTMLSelectElement).value = set
    }

    if ((select as HTMLSelectElement).value != set) {
      (select as HTMLSelectElement).value = "-"
    }

    return select
  }

  return
}

function getValueFromProviderFile(file: string, fileType, key) {

  if (file == "Threadfin Dummy") {
    return file
  }

  var fileID: string
  var indicator = file.charAt(0)

  switch (indicator) {
    case "M":
      fileType = "m3u"
      fileID = file
      break;

    case "H":
      fileType = "hdhr"
      fileID = file
      break;

    case "X":
      fileType = "xmltv"
      fileID = file.substring(0, file.lastIndexOf('.'))
      break;

  }

  if (SERVER["settings"]["files"][fileType].hasOwnProperty(fileID) == true) {
    var data = SERVER["settings"]["files"][fileType][fileID];
    return data[key]
  }

  return

}

function setXmltvChannel(id, element, dummy_type) {

  var xmltv: XMLTVFile = new XMLTVFile()
  var xmlFile = element.value

  var tvgId: string = SERVER["xepg"]["epgMapping"][id]["tvg-id"]
  var td = document.getElementById("popup-mapping").parentElement
  td.innerHTML = ""

  var select = xmltv.getPrograms(element.value, tvgId, false)
  select.setAttribute("name", "x-mapping")
  select.setAttribute("id", "popup-mapping")
  select.setAttribute("onchange", "javascript: this.className = 'changed'; checkXmltvChannel('" + id + "',this,'" + xmlFile + "');")
  select.className = "changed"
  sortSelect(select)
  td.appendChild(select);

  checkXmltvChannel(id, select, xmlFile)
}

function checkPPV(title, element) {
  var value = (element as HTMLSelectElement).value
  console.log("DUMMY TYPE: " + value)
  if(value == "PPV") {
    var td = document.getElementById("x-ppv-extra").parentElement
    td.innerHTML = ""

    var dbKey: string = "x-ppv-extra"
    var input = document.createElement("INPUT")
    input.setAttribute("type", "text")
    input.setAttribute("name", dbKey)
    // input.setAttribute("value", value)
    input.setAttribute("onchange", "javascript: this.className = 'changed'")
    input.setAttribute("id", "ppv-extra")
    
    var tr = document.createElement("TR")

    // Bezeichnung
    if (title.length != 0) {
      var td = document.createElement("TD")
      td.className = "left"
      td.innerHTML = title + ":"
    }


    // Content
    td.appendChild(element)
    this.table.appendChild(tr)
  }
}

function checkXmltvChannel(id: string, element: any, xmlFile) {

  var value = (element as HTMLSelectElement).value
  var bool: boolean
  var checkbox = document.getElementById('active')
  var channel: any = SERVER["xepg"]["epgMapping"][id]
  var updateLogo: boolean


  if (value == "-") {
    bool = false
  } else {
    bool = true
  }

  (checkbox as HTMLInputElement).checked = bool
  checkbox.className = "changed"
  console.log(xmlFile);

  // Kanallogo aktualisieren
  /*
  updateLogo = (document.getElementById("update-icon") as HTMLInputElement).checked
  console.log(updateLogo);
  */

  if (xmlFile != "Threadfin Dummy" && bool == true) {

    //(document.getElementById("update-icon") as HTMLInputElement).checked = true;
    //(document.getElementById("update-icon") as HTMLInputElement).className = "changed";

    console.log("ID", id)
    changeChannelLogo(id)

    return
  }

  if (xmlFile == "Threadfin Dummy") {
    (document.getElementById("update-icon") as HTMLInputElement).checked = false;
    (document.getElementById("update-icon") as HTMLInputElement).className = "changed";
  }

  return
}

function changeChannelLogo(id: string) {

  var updateLogo: boolean
  var channel: any = SERVER["xepg"]["epgMapping"][id]

  var f = (document.getElementById("popup-xmltv") as HTMLSelectElement);
  var xmltvFile = f.options[f.selectedIndex].value;

  var m = (document.getElementById("popup-mapping") as HTMLSelectElement);
  var xMapping = m.options[m.selectedIndex].value;

  var xmltvLogo = SERVER["xepg"]["xmltvMap"][xmltvFile][xMapping]["icon"]
  updateLogo = (document.getElementById("update-icon") as HTMLInputElement).checked

  if (updateLogo == true && xmltvFile != "Threadfin Dummy") {

    if (SERVER["xepg"]["xmltvMap"][xmltvFile].hasOwnProperty(xMapping)) {
      var logo = xmltvLogo
    } else {
      logo = channel["tvg-logo"]
    }

    var logoInput = (document.getElementById("channel-icon") as HTMLInputElement);
    logoInput.value = logo
    if (BULK_EDIT == false) {
      logoInput.className = "changed"
    }

  }

}

function savePopupData(dataType: string, id: string, remove: Boolean, option: number) {

  showElement("loading", true)

  if (dataType == "mapping") {


    var data = new Object()
    console.log("Save mapping data")

    cmd = "saveEpgMapping"
    data["epgMapping"] = SERVER["xepg"]["epgMapping"]

    console.log("SEND TO SERVER");

    var server: Server = new Server(cmd)
    server.request(data)

    delete UNDO["epgMapping"]

    showElement("loading", false)

    return

  }

  console.log("Save popup data")
  var div = document.getElementById("popup-custom")

  var inputs = div.getElementsByTagName("TABLE")[0].getElementsByTagName("INPUT");
  var selects = div.getElementsByTagName("TABLE")[0].getElementsByTagName("SELECT");

  var input = new Object();
  var confirmMsg: string

  for (let i = 0; i < selects.length; i++) {

    var name: string
    name = (selects[i] as HTMLSelectElement).name
    var value = (selects[i] as HTMLSelectElement).value

    switch (name) {
      case "tuner":
        input[name] = parseInt(value)
        break;

      default:
        input[name] = value
        break;
    }

  }

  for (let i = 0; i < inputs.length; i++) {

    switch ((inputs[i] as HTMLInputElement).type) {

      case "checkbox":
        name = (inputs[i] as HTMLInputElement).name
        input[name] = (inputs[i] as HTMLInputElement).checked
        break

      case "text":
      case "hidden":
      case "password":

        name = (inputs[i] as HTMLInputElement).name

        switch (name) {
          case "tuner":
            input[name] = parseInt((inputs[i] as HTMLInputElement).value)
            break;

          default:
            input[name] = (inputs[i] as HTMLInputElement).value
            break;
        }

        break

    }

  }

  var data = new Object()

  var cmd: string

  if (remove == true) {
    input["delete"] = true
  }

  switch (dataType) {
    case "users":

      confirmMsg = "Delete this user?"
      if (id == "-") {
        cmd = "saveNewUser"
        data["userData"] = input
      } else {
        cmd = "saveUserData"
        var d = new Object()
        d[id] = input
        data["userData"] = d
      }

      break;

    case "m3u":

      confirmMsg = "Delete this playlist?"
      switch (option) {
        // Popup: Save
        case 0:
          cmd = "saveFilesM3U"
          break

        // Popup: Update
        case 1:
          cmd = "updateFileM3U"
          break

      }

      data["files"] = new Object
      data["files"][dataType] = new Object
      data["files"][dataType][id] = input

      break

    case "hdhr":

      confirmMsg = "Delete this HDHomeRun tuner?"
      switch (option) {
        // Popup: Save
        case 0:
          cmd = "saveFilesHDHR"
          break

        // Popup: Update
        case 1:
          cmd = "updateFileHDHR"
          break

      }

      data["files"] = new Object
      data["files"][dataType] = new Object
      data["files"][dataType][id] = input

      break

    case "xmltv":

      confirmMsg = "Delete this XMLTV file?"
      switch (option) {
        // Popup: Save
        case 0:
          cmd = "saveFilesXMLTV"
          break

        // Popup: Update
        case 1:
          cmd = "updateFileXMLTV"
          break

      }

      data["files"] = new Object
      data["files"][dataType] = new Object
      data["files"][dataType][id] = input

      break

    case "filter":

      confirmMsg = "Delete this filter?"
      cmd = "saveFilter"
      data["filter"] = new Object
      data["filter"][id] = input
      break

    default:
      console.log(dataType, id);
      return
      break;

  }

  if (remove == true) {

    if (!confirm(confirmMsg)) {
      showElement("popup", false)
      return
    }

  }

  console.log("SEND TO SERVER");

  console.log(data);

  var server: Server = new Server(cmd)
  server.request(data)

  showElement("loading", false)

}

function donePopupData(dataType: string, idsStr: string) {

  var ids: string[] = idsStr.split(',');
  var div = document.getElementById("popup-custom")
  var inputs = div.getElementsByClassName("changed")

  ids.forEach(id => {
    var input = new Object();
    input = SERVER["xepg"]["epgMapping"][id]
    console.log("INPUT: " + input)

    for (let i = 0; i < inputs.length; i++) {

      var name: string
      var value: any

      switch (inputs[i].tagName) {

        case "INPUT":
          switch ((inputs[i] as HTMLInputElement).type) {
            case "checkbox":
              name = (inputs[i] as HTMLInputElement).name
              value = (inputs[i] as HTMLInputElement).checked
              input[name] = value
              break

            case "text":
              name = (inputs[i] as HTMLInputElement).name
              value = (inputs[i] as HTMLInputElement).value
              input[name] = value
              break

          }

          break

        case "SELECT":
          name = (inputs[i] as HTMLSelectElement).name
          value = (inputs[i] as HTMLSelectElement).value
          input[name] = value
          break

      }

      switch (name) {


        case "tvg-logo":
          //(document.getElementById(id).childNodes[2].firstChild as HTMLElement).setAttribute("src", value)
          break

        case "x-channel-start":
          (document.getElementById(id).childNodes[3].firstChild as HTMLElement).innerHTML = value
          break

        case "x-name":
          (document.getElementById(id).childNodes[3].firstChild as HTMLElement).innerHTML = value
          break

        case "x-category":
          var color = "white"
          var catColorSettings = SERVER["settings"]["epgCategoriesColors"]
            var colors_split = catColorSettings.split("|")
            for (var ii=0; ii < colors_split.length; ii++) {
              var catsColor_split = colors_split[ii].split(":")
              if (catsColor_split[0] == value) {
                color = catsColor_split[1]
              }
            }
          (document.getElementById(id).childNodes[3].firstChild as HTMLElement).style.borderColor = color
          break

        case "x-group-title":
          (document.getElementById(id).childNodes[5].firstChild as HTMLElement).innerHTML = value
          break

        case "x-xmltv-file":
          if (value != "Threadfin Dummy" && value != "-") {
            value = getValueFromProviderFile(value, "xmltv", "name")
          }

          if (value == "-") {
            input["x-active"] = false
          }

          (document.getElementById(id).childNodes[6].firstChild as HTMLElement).innerHTML = value
          break

        case "x-mapping":
          if (value == "-") {
            input["x-active"] = false
          }

          (document.getElementById(id).childNodes[7].firstChild as HTMLElement).innerHTML = value

          break

        case "x-backup-channel":
          (document.getElementById(id).childNodes[7].firstChild as HTMLElement).innerHTML = value

          break

        case "x-hide-channel":
          (document.getElementById(id).childNodes[7].firstChild as HTMLElement).innerHTML = value

          break

        default:

      }

      createSearchObj()
      searchInMapping()

    }

    if (input["x-active"] == false) {
      document.getElementById(id).className = "notActiveEPG"
    } else {
      document.getElementById(id).className = "activeEPG"
    }

    console.log(input["tvg-logo"]);
    (document.getElementById(id).childNodes[2].firstChild as HTMLElement).setAttribute("src", input["tvg-logo"])


  });

  showElement("popup", false);

  return
}

function showPreview(element: boolean) {

  var div = document.getElementById("myStreamsBox")
  switch (element) {

    case false:
      div.className = "notVisible"
      return
      break;
  }

  var streams: string[] = ["activeStreams", "inactiveStreams"]

  streams.forEach(preview => {

    var table = document.getElementById(preview)
    table.innerHTML = ""
    var obj: string[] = SERVER["data"]["StreamPreviewUI"][preview]

    var caption = document.createElement("CAPTION")
    var result = preview.replace( /([A-Z])/g, " $1" );
    var finalResult = result.charAt(0).toUpperCase() + result.slice(1);
    caption.innerHTML = finalResult
    table.appendChild(caption)

    var tbody = document.createElement("TBODY")
    table.appendChild(tbody)

    obj.forEach(channel => {

      var tr = document.createElement("TR")
      var tdKey = document.createElement("TD")
      var tdVal = document.createElement("TD")

      tdKey.className = "tdKey"
      tdVal.className = "tdVal"

      switch (preview) {
        case "activeStreams":
          tdKey.innerText = "Channel: (+)"
          break;

        case "inactiveStreams":
          tdKey.innerText = "Channel: (-)"
          break;
      }

      tdVal.innerText = channel
      tr.appendChild(tdKey)
      tr.appendChild(tdVal)

      tbody.appendChild(tr)

      table.appendChild(tr)

    });

  });

  // showElement("loading", false)
  div.className = "visible"

  return
}
