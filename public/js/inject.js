// here is a switch theme listener example
addThemeSwitchBroadcastListener(function(theme){
    console.log(theme);
});
// here is a context menu example
addContextMenuItem(function(event){
console.log("dicision function called",event)
return true;
},"test",function(event){
console.log("contextmenu called,event",event);
})
