class WizardCategory {
  DocumentID = "content"

  createCategoryHeadline(value:string):any {
    var element = document.createElement("H4")
    element.innerHTML = value
    return element
  }
}

class WizardItem extends WizardCategory {
  key:string
  headline:string

  constructor(key:string, headline:string) {
    super()
    this.headline = headline
    this.key = key
  }

  createWizard():void {
    var headline = this.createCategoryHeadline(this.headline)
    var key = this.key
    var content:PopupContent = new PopupContent()
    var description:string

    var doc = document.getElementById(this.DocumentID)
    doc.innerHTML = ""
    doc.appendChild(headline)

    switch (key) {
      case "tuner":
        var text = new Array()
        var values = new Array()

        for (var i = 1; i <= 100; i++) {
          text.push(i)
          values.push(i)
        }

        var select = content.createSelect(text, values, "1", key)
        select.setAttribute("class", "wizard")
        select.id = key
        doc.appendChild(select)

        description = "{{.wizard.tuner.description}}"

        break;
      
      case "epgSource":
        var text:any[] = ["PMS", "XEPG"]
        var values:any[] = ["PMS", "XEPG"]

        var select = content.createSelect(text, values, "XEPG", key)
        select.setAttribute("class", "wizard")
        select.id = key
        doc.appendChild(select)

        description = "{{.wizard.epgSource.description}}"

        break

      case "m3u":
        var input = content.createInput("text", key, "")
        input.setAttribute("placeholder", "{{.wizard.m3u.placeholder}}")
        input.setAttribute("class", "wizard")
        input.id = key
        doc.appendChild(input)

        description = "{{.wizard.m3u.description}}"

        break

      case "xmltv":
        var input = content.createInput("text", key, "")
        input.setAttribute("placeholder", "{{.wizard.xmltv.placeholder}}")
        input.setAttribute("class", "wizard")
        input.id = key
        doc.appendChild(input)

        description = "{{.wizard.xmltv.description}}"

      break

      default:
        console.log(key)
        break;
    }

    var pre = document.createElement("PRE")
    pre.innerHTML = description
    doc.appendChild(pre)

    console.log(headline, key)
  }


}


function readyForConfiguration(wizard:number) {

  var server:Server = new Server("getServerConfig")
  server.request(new Object())

  configurationWizard[wizard].createWizard()

}

function saveWizard() {

  var cmd = "saveWizard"
  var div = document.getElementById("content")
  var config = div.getElementsByClassName("wizard")

  var wizard = new Object()

  for (var i = 0; i < config.length; i++) {

    var name:string
    var value:any
    
    switch (config[i].tagName) {
      case "SELECT":
        name = (config[i] as HTMLSelectElement).name
        value = (config[i] as HTMLSelectElement).value

        // Wenn der Wert eine Zahl ist, wird dieser als Zahl gespeichert
        if(isNaN(value)){
          wizard[name] = value
        } else {
          wizard[name] = parseInt(value)
        }

        break

      case "INPUT":
        switch ((config[i] as HTMLInputElement).type) {
          case "text":
            name = (config[i] as HTMLInputElement).name
            value = (config[i] as HTMLInputElement).value

            if (value.length == 0) {
              var msg = name.toUpperCase() + ": " + "{{.alert.missingInput}}"
              alert(msg)
              return
            }

            wizard[name] = value
            break
        }
        break
      
      default:
        // code...
        break;
    }

  }

  var data = new Object()
  data["wizard"] = wizard

  var server:Server = new Server(cmd)
  server.request(data)

  console.log(data)
}

// Wizard
var configurationWizard = new Array()
configurationWizard.push(new WizardItem("tuner", "{{.wizard.tuner.title}}"))
configurationWizard.push(new WizardItem("epgSource", "{{.wizard.epgSource.title}}"))
configurationWizard.push(new WizardItem("m3u", "{{.wizard.m3u.title}}"))
configurationWizard.push(new WizardItem("xmltv", "{{.wizard.xmltv.title}}"))

// Show configuration progress indicator
function showConfigProgress(message: string, progress: number = 0): void {
    let configProgress = document.getElementById("config-progress");
    if (!configProgress) {
        // Create Bootstrap progress indicator in configuration header
        configProgress = document.createElement("div");
        configProgress.id = "config-progress";
        configProgress.className = "alert alert-primary alert-dismissible fade show";
        configProgress.style.cssText = "margin: 15px; border-radius: 8px; border-left: 4px solid #0d6efd;";
        
        configProgress.innerHTML = `
            <div class="d-flex align-items-center">
                <div class="spinner-border spinner-border-sm me-2" role="status">
                    <span class="visually-hidden">Processing...</span>
                </div>
                <div class="flex-grow-1">
                    <strong>Configuration Processing:</strong> ${message}
                    <div class="progress mt-2" style="height: 8px;">
                        <div class="progress-bar progress-bar-striped progress-bar-animated" 
                             role="progressbar" 
                             style="width: ${progress}%" 
                             aria-valuenow="${progress}" 
                             aria-valuemin="0" 
                             aria-valuemax="100">
                        </div>
                    </div>
                </div>
                <button type="button" class="btn-close ms-2" data-bs-dismiss="alert" aria-label="Close"></button>
            </div>
        `;
        
        // Insert at the top of the configuration content
        const configContent = document.querySelector('.container-fluid, .container, main, .content, .row') || document.body;
        if (configContent.firstChild) {
            configContent.insertBefore(configProgress, configContent.firstChild);
        } else {
            configContent.appendChild(configProgress);
        }
    } else {
        // Update existing progress
        const progressBar = configProgress.querySelector('.progress-bar') as HTMLElement;
        const messageDiv = configProgress.querySelector('.flex-grow-1 strong');
        if (progressBar) {
            progressBar.style.width = `${progress}%`;
            progressBar.setAttribute('aria-valuenow', progress.toString());
        }
        if (messageDiv && messageDiv.nextSibling) {
            messageDiv.nextSibling.textContent = ` ${message}`;
        }
    }
}

// Update configuration progress
function updateConfigProgress(progress: number): void {
    const configProgress = document.getElementById("config-progress");
    if (configProgress) {
        const progressBar = configProgress.querySelector('.progress-bar') as HTMLElement;
        if (progressBar) {
            progressBar.style.width = `${progress}%`;
            progressBar.setAttribute('aria-valuenow', progress.toString());
        }
    }
}

// Hide configuration progress
function hideConfigProgress(): void {
    const configProgress = document.getElementById("config-progress");
    if (configProgress) {
        // Use Bootstrap's fade out animation
        configProgress.classList.remove('show');
        configProgress.classList.add('fade');
        setTimeout(() => {
            if (configProgress.parentNode) {
                configProgress.parentNode.removeChild(configProgress);
            }
        }, 150);
    }
}



