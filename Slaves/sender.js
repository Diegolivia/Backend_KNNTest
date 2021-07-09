function SubmitForm() {

    let name=document.getElementById("name")
    let email=document.getElementById("email")

    let url = "http://localhost:6969/data"
    var xhr = new XMLHttpRequest();

    xhr.open("POST",url,true)
    xhr.setRequestHeader("Content-Type", "application/json");
    var data = JSON.stringify({ "name": name.value, "email": email.value });
    console.log(name.value)
    console.log(email.value)
    xhr.send(data)
  }
  