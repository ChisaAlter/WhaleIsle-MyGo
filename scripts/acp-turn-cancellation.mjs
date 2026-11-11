// Native Host cancellation owns the external prompt too. A later turn waits
// for T3 to release that prompt instead of reading its abandoned queue.
export function bindTurnCancellation(turn, api) {
  const signal=turn.signal;
  if(!signal)return;
  const connection=turn.connection;
  const abort=()=>{
    if(turn.promptDone||connection.pendingCancellation)return;
    const pending=Promise.resolve()
      .then(()=>api.cancel())
      .then(()=>turn.promptPromise)
      .catch(async error=>{await connection.close();throw error;})
      .finally(()=>{
        if(connection.activeTurn===turn)connection.activeTurn=null;
        if(connection.pendingCancellation===pending)connection.pendingCancellation=null;
      });
    connection.pendingCancellation=pending;
    pending.catch(()=>{}); // consumed by the next stream; never an unhandled rejection
  };
  signal.addEventListener('abort',abort,{once:true});
  turn.promptPromise.finally(()=>signal.removeEventListener('abort',abort)).catch(()=>{});
  if(signal.aborted)abort();
}
