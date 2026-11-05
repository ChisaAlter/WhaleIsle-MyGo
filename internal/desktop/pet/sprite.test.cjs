"use strict";
const {test}=require('node:test'),assert=require('node:assert/strict');
const {cell}=require('./assets/pet-sprite.js');
test('full atlas rows and clockwise gaze directions reach the actual crop selection',()=>{
  for(const [state,row] of [['sleep',6],['angry',5],['pick-up',4],['eat',4],['celebrate',4],['react-head',3],['run-left',2],['run-right',1]])assert.equal(cell(11,3,state,{},null).row,row);
  assert.deepEqual(cell(9,2,'',{},null),{row:0,col:2});
  assert.equal(cell(11,2,'',{State:'running'},null).row,7);
  assert.equal(cell(11,2,'',{Look:{Pinned:true}},null).row,8);
  for(let index=0;index<16;index++){const angle=index*Math.PI/8;assert.deepEqual(cell(11,0,'',{}, {x:Math.sin(angle),y:-Math.cos(angle)}),{row:index<8?9:10,col:index%8});}
  assert.equal(cell(9,0,'',{}, {x:1,y:0}).row,0);
});
