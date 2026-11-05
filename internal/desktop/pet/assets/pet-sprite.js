"use strict";
// Codex community-atlas contract: 8x9 rows, with 16 clockwise gaze cells
// appended by v2. Direction zero points up, not to the right.
(function(root){
 function cell(rows,frame,state,task,gaze){
  let row=0,col=frame%8;
  if(state==='sleep'||task?.Look?.Busy){row=6;col=0}
  else if(state==='angry'||task?.NotificationState==='error')row=5;
  else if(['pick-up','eat','celebrate'].includes(state))row=4;
  else if(state==='react-head')row=3;
  else if(state==='run-left')row=2;
  else if(state==='run-right')row=1;
  else if(task?.Look?.Pinned||task?.NotificationState==='review')row=8;
  else if(task?.State==='running')row=7;
  else if(rows===11&&gaze){const degrees=Math.atan2(gaze.x,-gaze.y)*180/Math.PI;const index=Math.round(((degrees+360)%360)/22.5)%16;row=index<8?9:10;col=index%8}
  return {row,col};
 }
 root.WhalePetSprite={cell};if(typeof module==='object')module.exports={cell};
})(typeof window==='object'?window:globalThis);
