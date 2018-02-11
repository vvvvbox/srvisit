	function getXmlHttp(){
		var xmlhttp;
		try {
			xmlhttp = new ActiveXObject("Msxml2.XMLHTTP");
			} catch (e) {
			try {
				xmlhttp = new ActiveXObject("Microsoft.XMLHTTP");
				} catch (E) {
					xmlhttp = false;
					}
				}
		if (!xmlhttp && typeof XMLHttpRequest!='undefined') {
			xmlhttp = new XMLHttpRequest();
			}
		return xmlhttp;
	}
	
	function loadMenu(){
		for(i = menu.length - 1; i >= 0; i--){
			var newA = document.createElement('a');
			newA.setAttribute('href', menu[i].Link);
			newA.innerHTML = menu[i].Capt;
			
			document.getElementById('menu').appendChild(newA);
		}
	}
			
	
	function changeFunc(select){
		if (select.value == "newest" || select.value == "available") {
			document.getElementById('iperiod').style.display = "inline-block";
			document.getElementById('lperiod').style.display = "inline-block";
		}else{
			document.getElementById('iperiod').style.display = "none";
			document.getElementById('lperiod').style.display = "none";
		}
		if (select.value == "newest") {
			document.getElementById('iperiod').placeholder = 'Таймаут в часах';
			document.getElementById('lperiod').innerHTML = 'Таймаут в часах';
		}
		if (select.value == "available") {
			document.getElementById('iperiod').placeholder = 'Размер в МБ';
			document.getElementById('lperiod').innerHTML = 'Свободное место в МБ';
		}
	}
	
	function getDialogServer(){
		var r = "<div style='margin-top: !1'>".replace('!1',  (window.pageYOffset + 100) + 'px') +
				"<div class='header'>Ресурс для проверки</div>" +
				"<form action='/tester' >" +
				"<div class='content'>" +

				"<input type='hidden' name='make' value='add'>" +
				
				"<span>Название ресурса</span><br>" +
				"<input type='text' placeholder='название'  name='name'>" +

				"<span>Адрес ресурса</span><br>" +
				"<input type='text' placeholder='адрес'  name='addr'>" +
				
				"</div>" +
				"<input type='button' class='button' value='Отмена' onclick='closeDialog(this)'>" +
				"<input type='submit' class='button' value='Добавить'>" +
				"</form>" +
				"</div>";
		return r;
	}
	
	function getDialogPhone(){
		var r = "<div style='margin-top: !1'>".replace('!1',  (window.pageYOffset + 100) + 'px') +
				"<div class='header'>Телефон для оповещения</div>" +
				"<form action='/tester' >" +
				"<div class='content'>" +

				"<input type='hidden' name='make' value='addt'>" +
				
				"<span>Номер телефона</span><br>" +
				"<input type='text' placeholder='номер телефона'  name='tel'>" +

				"</div>" +
				"<input type='button' class='button' value='Отмена' onclick='closeDialog(this)'>" +
				"<input type='submit' class='button' value='Добавить'>" +
				"</form>" +
				"</div>";
		return r;
	}

	function getDialogOption(id, note, timeout){
		var r = "<div style='margin-top: !1'>".replace('!1',  (window.pageYOffset + 100) + 'px') +
				"<div class='header'>Опции агента</div>" +
				"<form action='/agents' >" +
				"<div class='content'>" +

				"<input type='hidden' name='make' value='editagent'>" +
				"<input type='hidden' name='name' value='!1'>".replace('!1',  id) +

				"<span>Название</span><br>" +
				"<input type='text' placeholder='Введите название' value='!1' name='note'>".replace('!1', note) +

				"<span>Таймаут связи в минутах</span><br>" +
				"<input type='text' placeholder='таймаут' value='!1' name='timeout'>".replace('!1',  timeout) +

				"</div>" +
				"<input type='button' class='button' value='Отмена' onclick='closeDialog(this)'>" +
				"<input type='submit' class='button' value='Сохранить'>" +
				"</form>" +
				"</div>";
		return r;
	}

	function getDialogAdd(id){
		var r = "<div style='margin-top: !1'>".replace('!1',  (window.pageYOffset + 100) + 'px') +
				"<div class='header'>Добавить агент</div>" +
				"<form action='/agents' >" +
				"<div class='content'>" +

				"<input type='hidden' name='make' value='addtest'>" +
				"<input type='hidden' name='name' value='!1'>".replace('!1',  id) +

				"<span>Название</span><br>" +
				"<input type='text' placeholder='Введите название' name='note'>" +

				"<span>Объект проверки</span><br>" +
				"<input type='text' placeholder='Объект проверки' name='resource'>" +

				"<span>Тип проверки</span><br>" +
				"<select name='type' name='type' onchange='changeFunc(this);'>" +
				"	<option value='ping'>Доступность ресурса</option>" +
				"	<option value='exists'>Доступность файла</option>" +
				"	<option value='exec'>Выполнение команды</option>" +
				"	<option value='newest'>Самый новый файл</option>" +
				"	<option value='available'>Свободное место</option>" +
				"</select>" +

				"<span style='display: none;' id='lperiod'>Таймаут проверки</span><br>" +
				"<input type='text' style='display: none;' id='iperiod' name='period' placeholder='Таймаут в часах'>" +


				"</div>" +
				"<input type='button' class='button' value='Отмена' onclick='closeDialog(this)'>" +
				"<input type='submit' class='button' value='Добавить'>" +
				"</form>" +
				"</div>";
		return r;
	}

	function showDialog(body){
		var d = document.documentElement;
		//d.style.overflow = 'hidden';
		var scrollHeight = Math.max(
		  document.body.scrollHeight, document.documentElement.scrollHeight,
		  document.body.offsetHeight, document.documentElement.offsetHeight,
		  document.body.clientHeight, document.documentElement.clientHeight
		);

		document.onkeydown = function(evt) {
			evt = evt || window.event;
			var isEscape = false;
			if ("key" in evt) {
				isEscape = (evt.key == "Escape" || evt.key == "Esc");
			} else {
				isEscape = (evt.keyCode == 27);
			}
			if (isEscape) {
				closeDialog();
			}
		};

		var v = document.createElement('div');


		v.style.width = "100%";
		v.style.height = scrollHeight + "px";
		v.setAttribute('class', 'dialog');

		v.innerHTML = body;
		d.appendChild(v);
	}

	function closeDialog(dialog){
		document.onkeydown = '';
		var d = document.documentElement;
		//d.style.overflow = 'auto';
		var e = document.getElementsByClassName('dialog');
		e[0].remove();
		//dialog.parentNode.remove(0);
	}