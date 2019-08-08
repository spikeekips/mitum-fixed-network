import uuidv4 from 'uuid/v4'

class Time {
  constructor() {
    this.t = null
    this.n = null
    this.orig = null
  }

  static parse(s) {
    var iso0 = /(\d{4})-([01]\d)-([0-3]\d)T([0-2]\d):([0-5]\d):([0-5]\d)\.(\d+)([+-][0-2]\d:[0-5]\d|Z)/
    var iso1 = /(\d{4})-([01]\d)-([0-3]\d)T([0-2]\d):([0-5]\d):([0-5]\d)([+-][0-2]\d:[0-5]\d|Z)/

    var p = s.match(iso0);
    if (p == null) {
      p = s.match(iso1)
      if (p == null) {
        throw new Error('invalid time: ' + s)
      }
    }

    var t = new Date(
      p[1],
      Number.parseInt(p[2], 10) - 1,
      p[3],
      p[4],
      p[5],
      p[6],
    )

    var tail = p[7].length > 0 ? p[7] : 0
    if (tail.length < 6) {
      tail = tail + '0'.repeat(6 - tail.length)
    }

    var time = new Time()
    time.t = t
    time.n = (t.getTime() * 1000000) + Number.parseInt(tail, 10)
    time.orig = s

    return time
  }

  elapsed(t) {
    var d = this.n - t.n
    if (d < 0) {
      return '000.000000s'
    }

    var s = parseInt(d / 1000000000)
    var head = s.toString()
    if (head.length < 3) {
      head = '0'.repeat(3 - head.length) + head
    }

    var tail = (d- (s * 1000000000)).toString()
    if (tail.length < 6) {
      tail = '0'.repeat(6 - tail.length) + tail
    }

    return head + '.' + tail + 's'
  }
}

class Record {
  constructor() {
    this.module = null
    this.message = null
    this.level = null
    this.t = null
    this.node = null
    this.caller = null
    this.extra = null
    this.body = null
    this.id = null
  }

  static fromJSONString(line) {
    var o = null
    try {
      o = JSON.parse(line)
    } catch(e) {
      console.warn(e)
      return null
    }

    // module
    var module = null
    if ('module' in o === false) {
      throw new Error('module is missing')
    } else {
      module = o['module']
    }

    // message
    var message = null
    if ('msg' in o === false) {
      throw new Error('message is missing')
    } else {
      message = o['msg']
    }

    // level
    var level = null
    if ('lvl' in o === false) {
      throw new Error('level is missing')
    } else {
      level = o['lvl']
    }

    // t
    var t = null
    var id = null
    if ('t' in o === false) {
      throw new Error('t is missing')
    } else {
      t = Time.parse(o['t'])
      id = t.n + '-' + uuidv4()
    }

    // node
    var node = null
    if ('node' in o === false) {
      //
    } else {
      node = o['node']
    }

    // caller
    var caller = null
    if ('caller' in o === false) {
      throw new Error('caller is missing')
    } else {
      caller = o['caller']
    }

    // extra
    var extra = null;
    delete o.msg
    delete o.lvl
    delete o.t
    delete o.caller
    delete o.node
    delete o.module

    extra = o;

    // body
    var r = new Record()

    r.id = id
    r.t = t
    r.module = module
    r.message = message
    r.level = level
    r.node = node
    r.caller = caller
    r.extra = extra
    r.body = line;

    return r
  }

  basic() {
    return {
      t: this.t,
      module: this.module,
      message: this.message,
      level: this.level,
      node: this.node,
      caller: this.caller,
      body: this.line,
    }
  }
}

class Log {
  constructor() {
    this.nodes = []
    this.records = []
    this.msgs = []
    this.levels = []
    this.modules = []
  }

  static load (contents) {
    var log = new Log()

    var records = []
    var nodes = []
    var msgs = []
    var levels = []
    var modules = []
    var line = ''
    for (const c of contents) {
      if (c === '\n') {
        var record = null
        record = this.parseRecord(line)
        line = ''

        if (record === undefined) {
          continue
        }

        // node
        if (record.node == null) {
          continue
        } else if (!nodes.includes(record.node)) {
          nodes.push(record.node)
        }

        records.push(record)
        if (!msgs.includes(record.message)) {
          msgs.push(record.message)
        }

        if (!levels.includes(record.level)) {
          levels.push(record.level)
        }

        if (!modules.includes(record.module)) {
          modules.push(record.module)
        }

        continue
      }

      line += c
    }

    records.sort(function(a, b) {
      return a.t.n - b.t.n
    });

    msgs.sort();

    nodes.sort()
    log.nodes = nodes
    log.records = records
    log.msgs = msgs
    log.levels = levels
    log.modules = modules

    return log
  }

  static parseRecord(line) {
    var record = null
    try {
      record = Record.fromJSONString(line)
    } catch (e) {
      console.error(e)
      return
    }

    if (record == null || record.node == null) {
      return
    }

    return record
  }
}

export default Log
//module.exports = Log
