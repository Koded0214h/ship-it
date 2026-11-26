declare const require: any, process: any
require('express')().get('/', (_: any, res: any) => res.send('ok')).listen(process.env.PORT || 3000, '0.0.0.0')
