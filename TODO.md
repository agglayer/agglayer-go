# TODOs

- Setup defaults for testing
- Etherman has a lot of methods that are copy/pasted from Etherman from zkevm-node, would be nice to decouple those so we can re-use
- Go stateless (don't require full nodes)
- decouple JSON RPC, also needed for DAC
- decouple tx tracing for users from the DB table of the eth tx man => more flexibility and control
    - What to do on already exisitng txs? Controled error? Just say OK?
- Run ethtxman in a separate process, so we can scale the RPC horizontaly
    - Will be important once we have state-less executor and have to re-process batches
- Improve rpc ctx
- Imporve logging
- Add GHA:
    - build & push docker images
    - lint
    - test