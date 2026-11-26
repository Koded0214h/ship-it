const port = process.env.PORT || 3000
require('express')().get('/', (_, res) => res.send('ok')).listen(port, '0.0.0.0')
