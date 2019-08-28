import React from 'react'
import ReactDOM from 'react-dom';
import './App.css';
import { withStyles } from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography';
import AttachFileIcon from '@material-ui/icons/AttachFile';
import Dropzone from 'react-dropzone'
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import Grid from '@material-ui/core/Grid';
import TableRow from '@material-ui/core/TableRow';
import { SnackbarProvider, withSnackbar } from 'notistack';
import SpeedDial from '@material-ui/lab/SpeedDial';
import SpeedDialIcon from '@material-ui/lab/SpeedDialIcon';
import IconButton from '@material-ui/core/IconButton';
import ChildCareIcon from '@material-ui/icons/ChildCare';
import BookmarksIcon from '@material-ui/icons/Bookmarks';
import SpeedDialAction from '@material-ui/lab/SpeedDialAction';
import Box from '@material-ui/core/Box';
import Chip from '@material-ui/core/Chip';
import Highlight from 'react-highlight'
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import Button from '@material-ui/core/Button';
import TextField from '@material-ui/core/TextField';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Checkbox from '@material-ui/core/Checkbox';
import FormControl from '@material-ui/core/FormControl';
import FormLabel from '@material-ui/core/FormLabel';
import FilterListIcon from '@material-ui/icons/FilterList';
import SettingsOverscanIcon from '@material-ui/icons/SettingsOverscan';
import stringify from 'csv-stringify';
import colormap from 'colormap';
import markdown from 'markdown';

import Log from './log'
import LogFilter from './filter'
import raw from './raw'

const styles = theme => ({
  root: {
    flexGrow: 1,
  },
  grow: {
    flexGrow: 1,
  },
  menuButton: {
    marginLeft: -12,
    marginRight: 20,
  },
  list: {
    width: '100%',
    overflowX: 'auto',
  },
  speedDial: {
    zIndex: 100000,
    position: 'fixed',
    bottom: theme.spacing(2),
    right: theme.spacing(3),
  },
  textField: {
    minWidth: '400px'
  }
});

var allLevels = ['error', 'debug', 'info', 'warn', 'fatal', 'panic']

var hexToRgb = (hex) => {
  var shorthandRegex = /^#?([a-f\d])([a-f\d])([a-f\d])$/i;
  hex = hex.replace(shorthandRegex, function(m, r, g, b) {
    return r + r + g + g + b + b;
  });

  var result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result ? [
    parseInt(result[1], 16),
    parseInt(result[2], 16),
    parseInt(result[3], 16),
  ] : null;
}

var fontColorByBG = (hex) => {
  var rgb = hexToRgb(hex)
  var o = Math.round(
    ((parseInt(rgb[0]) * 299) +
    (parseInt(rgb[1]) * 587) +
    (parseInt(rgb[2]) * 114)) / 1000)

  return (o > 140) ? 'black' : 'white'
}

class CenteredGrid extends React.Component {
  state = {
    menu: false,
    fileDialog: false,
    bottom: true,
    records: [],
    nodes: [],
    modules: [],
    record: null,
    speedDial: false,
    openDialog: false,
    moduleColors: [],
    filteredLevels: { error: true, debug: true, info: true, warn: true, fatal: true, panic: true},
  }

  log = null
  prevRecords = []
  prevRecordsFragment = []
  timeSpanOneRow = 8000000
  logFilter = null

  toggleDrawer = (open) => () => {
    this.setState({
      'menu': open,
    });
  }

  toggleBottom = (open) => () => {
    this.setState({
      'bottom': open,
    });
  }

  onSelectedFile = (acceptedFiles) => {
    this.headerResized = false
    var promises = []
    for (let file of acceptedFiles) {
      var p = new Promise(function(resolve, reject) {
        var reader = new FileReader();
        reader.onload = () => {
          resolve(reader.result)
        }

        reader.readAsBinaryString(file)
      })
      promises.push(p)
    }

    Promise.all(promises).then(values => {
      var result = ''.concat(...values)

      try {
        this.log = Log.load(result)
      } catch(e) {
        this.props.enqueueSnackbar('failed to load logs', {variant: 'error'})
        return
      }
      // initialize
      this.logFilter = new LogFilter(this.log.records, {})
      this.prevRecords = []
      this.prevRecordsFragment = []

      this.props.enqueueSnackbar(
        'logs successfully imported: ' + this.log.records.length + ' records found',
        {variant: 'info'},
      )

      var colors = colormap({
        colormap: 'hsv',
        nshades: this.log.modules.length > 10 ? this.log.modules.length : 11,
        format: 'hex',
        alpha: 1,
      })

      this.setState({
                records: [],
        nodes: this.log.nodes,
        levels: this.log.levels,
        modules: this.log.modules,
        moduleColors: colors,
      })

      this.renderRecordsMore()
    })
  }

  handleSpeedDialOpen = () => {
    this.setState({ speedDial: true, });
  };

  handleSpeedDialClose = () => {
    this.setState({ speedDial: false});
  };

  toggleExpandAll = () => {
    const node = ReactDOM.findDOMNode(this)
    if (! node instanceof HTMLElement) {
      return
    }

    const children = node.querySelectorAll('.row-detail')
    Array.from(children).map(c => {
      c.classList.toggle('row-detail-open')
      return null
    })
  }

  filters = {
    message: null,
    regexp: null,
  }

  onFilterFormChanged (name, value) {
    console.log('>', name, value, this.filters)

    switch (name) {
      case 'log-level-add':
        var f = this.state.filteredLevels
        f[value] = true
        this.setState({filteredLevels: f})
        break
      case 'log-level-remove':
        f = this.state.filteredLevels
        f[value] = false
        this.setState({filteredLevels: f})
        break
      case 'message':
        if (value.trim().length < 1) {
          this.filters.message = ""
          this.filters.regexp = null
          this.filters.message_error = null
        } else {
          if (value.search(/^\//) < 0) {
            this.filters.regexp = new RegExp(value)
          } else {
            try {
              this.filters.regexp = new RegExp(eval(value))
            } catch (e) {
              console.error(e)
              return null
            }
          }

          this.filters.message = value.trim()
        }
        break
      default:
        return null
    }

    var levels = []
    for (const [level, v] of Object.entries(this.state.filteredLevels)) {
      if (!v) {
        continue
      }
      levels.push(level)
    }

    this.logFilter.setFilters({levels: levels, message: this.filters.message})

    console.log('<', name, value, this.filters)
  }

  filterRecords (records) {
    var levels = []
    for (const [level, v] of Object.entries(this.state.filteredLevels)) {
      if (!v) {
        continue
      }
      levels.push(level)
    }

    console.log('filteredLevels:', levels)

    return records.filter(r => {
      if (levels.length > 0) {
        if (!levels.includes(r.level)) {
          return false
        }
      }

      if (this.filters.regexp !== null) {
        if (!this.filters.regexp.test(r.body)) {
          return false
        }
      }

      return true
    })
  }

  filter() {
    setTimeout(() => {
      this.setState({speedDial: false})
    }, 10);

    this.setState({ openDialog: false});
    console.log('filters:', this.filters)

    if (this.log === null || this.log.records === null) {
      return
    }

    this.logFilter.reset()
    this.prevRecords = []
    this.prevRecordsFragment = []
    var records = []
    for (let record of this.logFilter.filter(this.limit)) {
      records.push(record)
    }
    console.log('log filters:', this.logFilter.filters)

    this.props.enqueueSnackbar(
      'logs successfully filtered: ' + records.length + ' records found',
      {variant: 'info'},
    )

    this.setState({records: records})
  }

  handleCloseDialog = () => {
    this.setState({ openDialog: false});
  };

  importTestData = () => {
    this.log = Log.load(raw)
    this.logFilter = new LogFilter(this.log.records, {})
    this.prevRecords = []
    this.prevRecordsFragment = []

    this.props.enqueueSnackbar(
      'test log data successfully imported: ' + this.log.records.length + ' records found',
      {variant: 'info'},
    )

    var colors = colormap({
      colormap: 'hsv',
      nshades: this.log.modules.length > 10 ? this.log.modules.length : 11,
      format: 'hex',
      alpha: 1,
    })

    this.setState({
      nodes: this.log.nodes,
      levels: this.log.levels,
      modules: this.log.modules,
      moduleColors: colors,
    })

    this.renderRecordsMore()
  }

  exportToCSV() {
    if (this.log == null) {
      console.error('read log first')
      return
    }

    var data = []
    const stringifier = stringify({
      delimiter: ',',
      escape: "\\",
    })
    stringifier.on('readable', function(){
      let row;
      while(row = stringifier.read()){ // eslint-disable-line
        data.push(row)
      }
    })
    stringifier.on('error', function(err){
      console.error(err.message)
    })
    stringifier.on('finish', function(){
      var csvData = new Blob([data.join('')], {type: 'text/csv'})
      var csvurl = URL.createObjectURL(csvData)

      var link = document.createElement('a');
      link.href = csvurl;
      link.download='mitum-log-' + (new Date()).toISOString().replace(/[:.]/g, '-') + '.csv';
      link.click();
    })

    var header = ['t']
    header.push(...this.log.nodes)

    stringifier.write(header)

    this.state.records.map(record => {
      var row = [record.t.orig]

      this.log.nodes.map(node => {
        if (node === record.node) {
          row.push(JSON.stringify(record).replace(/\n/, ""))
        } else {
          row.push('')
        }
        return null
      })

      stringifier.write(row)
      return null
    })

    stringifier.end()
  }

  componentDidMount() {
    window.addEventListener('scroll', this.onScroll, false);
    //this.importTestData()
  }

  componentWillUnmount() {
    window.removeEventListener('scroll', this.onScroll, false);
  }

  onLoading = false
  limit = 500
  recordsOffset = 0

  onScroll = () => {
    var root = document.getElementById('inner-root')
    if ((window.scrollY + document.body.offsetHeight) >= (root.offsetHeight - 1)) {
      if (this.onLoading) {
        return
      }

      this.onLoading = true
      this.renderRecordsMore();
    }
  }

  toggleDetail(ref, open) {
    const tr = ReactDOM.findDOMNode(ref.current)

    if (open === undefined) {
      open = !tr.classList.contains('row-detail-open')
    }

    if (open === true) {
      tr.classList.add('row-detail-open')
    } else {
      tr.classList.remove('row-detail-open')
    }

    ref.current.toggle(open)

    return
  }

  sanitizeRecordMessage(message) {
    var a = markdown.markdown.toHTML(message)
    return a.slice(3, a.length - 4)
  }

  renderRecord(index, first, records, nodes) {
    const { classes } = this.props

    var early = records.filter(r => r != null).sort((a, b) => a.t.n - b.t.n)[0]
    if (early == null) {
      return
    }

    var rowRef = React.createRef()

    var rowDetailRefs = new Array(nodes.length)
    records.map((record, i) => {
      if (record == null) {
        return null
      }
      rowDetailRefs[i] = React.createRef()
      return null
    })

    return <React.Fragment key={index + 'f'}>
      <TableRow key={index} ref={rowRef}>
        <TableCell key={index + '-t'}>
          <IconButton className={classes.button} aria-label='Bookmark' onClick={e => {
            const tr = ReactDOM.findDOMNode(rowRef.current)
            tr.classList.toggle('selected')

            // find details
            var s = tr.nextSibling
            while(s.classList.contains('row-detail')) {
              s.classList.toggle('selected')
              s = s.nextSibling
            }
            rowDetailRefs.map(ref => this.toggleDetail(ref, true))
          }}>
            <BookmarksIcon />
          </IconButton>
          <span className='t'>
            {early.t.elapsed(first.t)}
          </span>
        </TableCell>
        {nodes.map((node, ni) => (
          <TableCell
            className={classes.listTableTd}
            key={ni + node + '-m'}
            onClick={e => {
              if (rowDetailRefs[ni] == null) {
                return
              }

              this.toggleDetail(rowDetailRefs[ni])
            }}
          >
          {records[ni] != null ? (
            <div key={records[ni].id + records[ni].module} className='record'>
              <Chip label={records[ni].level[0]} className={'lvl lvl-' + records[ni].level} color='secondary' />
              <Chip label={records[ni].module} className={'module'} color='secondary' style={{
                backgroundColor: this.state.moduleColors[this.state.modules.indexOf(records[ni].module)],
                color: fontColorByBG(this.state.moduleColors[this.state.modules.indexOf(records[ni].module)]),
              }} />
              <span dangerouslySetInnerHTML={{__html: this.sanitizeRecordMessage(records[ni].message)}} />
            </div>
            ) : (
              <Typography key={ni + node + 'ty'}></Typography>
            )
          }
          </TableCell>
        ))}
      </TableRow>

      {records.map((record, i) => {
        return <RecordDetail
          key={'rd' + index + '-' + i}
          classes={classes} nodes={nodes} record={record} ref={rowDetailRefs[i]} />
      })}

    </React.Fragment>
  }

  renderRecordsMore() {
        console.log('log filters:', this.logFilter.filters)

        var records = [
        ...this.state.records,
        ...this.logFilter.filter(this.limit),
        ]

    this.setState({ records: records, speedDial: false })

    this.recordsOffset += this.limit
    this.onLoading = false
  }

  renderRecords(records, nodes) {
    var update = false
    if (records.length !== this.prevRecords.length) {
      update = true
    } else if (records.length > 0 && this.prevRecords.length > 0) {
      var pl = this.prevRecords[this.prevRecords.length - 1].t.n
      var tl = records[records.length - 1].t.n
      update = pl !== tl
    }

    if (!update) {
      return this.prevRecordsFragment
    }

    this.prevRecords = records

    var rs = new Array(nodes.length)
    var last = null

    var index = this.prevRecordsFragment.length
    var updated = records.slice(this.prevRecordsFragment.length).map((record) => {
      var i = nodes.indexOf(record.node)
      if (rs[i] != null) {
        var o = this.renderRow(index, this.log.records[0], rs, nodes)
        index++

        rs = new Array(nodes.length)
        rs[i] = record
        last = record.t.n

        return o
      }

      if (last != null) {
        var sub = record.t.n - last
        if (sub > this.timeSpanOneRow) {
          o = this.renderRow(index, this.log.records[0], rs, nodes)
          index++

          rs = new Array(nodes.length)
          rs[i] = record
          last = record.t.n

          return o
        }
      }

      rs[i] = record
      last = record.t.n

      return null
    })

    updated.push(this.renderRow(records.length, this.log.records[0], rs, nodes))
    this.prevRecordsFragment = [
        ...this.prevRecordsFragment,
        ...updated,
    ]

    return this.prevRecordsFragment
  }

  shouldComponentUpdate(nextProps, nextState) {
    return true
  }

  renderRow(index, first, records, nodes) {
    const { classes } = this.props;

    if (index % 30 === 0) {
      return <React.Fragment key={'r' + index}>
        <TableRow className='header' key={'h' + index}>
          <TableCell className={classes.listTableT} key={'t'}><div>T</div></TableCell>
          {this.state.nodes.map(node => (
            <TableCell align='left' key={node}><div>{node}</div></TableCell>
          ))}
        </TableRow>
        {this.renderRecord(index, first, records, nodes)}
      </React.Fragment>
    }
    
    return this.renderRecord(index, first, records, nodes)
  }

  filterrMessageRef = React.createRef()
  render() {
    const { classes } = this.props;

    return <div className={classes.root}>
      <div className={classes.root}>
        <div style={{display: 'none'}}>
          <Dropzone ref='dropzone' onDrop={acceptedFiles => this.onSelectedFile(acceptedFiles)}>
            {({getRootProps, getInputProps}) => (
              <section>
                <div {...getRootProps()}>
                  <input {...getInputProps()} />
                  <div>Drag 'n' drop some files here, or click to select files</div>
                </div>
              </section>
            )}
          </Dropzone>
        </div>
      </div>

      <Box height='100%'>
        <Table id='inner-root' className={' scrollable'}>
          <TableBody>
            {this.renderRecords(this.state.records, this.state.nodes)}
          </TableBody>
        </Table>
      </Box>

      <SpeedDial
        ariaLabel='SpeedDial tooltip example'
        className={classes.speedDial}
        icon={<SpeedDialIcon />}
        onBlur={this.handleSpeedDialClose}
        onClick={this.handleSpeedDialClick}
        onClose={this.handleSpeedDialClose}
        onFocus={this.handleSpeedDialOpen}
        onMouseEnter={this.handleSpeedDialOpen}
        onMouseLeave={this.handleSpeedDialClose}
        open={this.state.speedDial}
      >
        <SpeedDialAction
          key={'import-log-file'}
          icon={<AttachFileIcon />}
          tooltipTitle={'import new log'}
          tooltipOpen
          onClick={e=>{this.refs['dropzone'].open()}}
        />
      {this.state.records.length > 0 ? (
        <SpeedDialAction
          key={'expand-collapse-all'}
          icon={<SettingsOverscanIcon />}
          tooltipTitle={'expand/collapse all'}
          tooltipOpen
          onClick={e=>{this.toggleExpandAll()}}
        />) : ([]) }
      <SpeedDialAction
        key={'filter'}
        icon={<FilterListIcon />}
        tooltipTitle={'filter records'}
        tooltipOpen
        onClick={e=>{this.setState({ openDialog: true})}}
      />
      {this.state.records.length > 0 ? (
        <SpeedDialAction
          key={'export to csv'}
          icon={<ChildCareIcon />}
          tooltipTitle={'export to csv'}
          tooltipOpen
          onClick={e=>{this.exportToCSV()}}
        />) : ([]) }
        <SpeedDialAction
          key={'test data'}
          icon={<ChildCareIcon />}
          tooltipTitle={'test data'}
          tooltipOpen
          onClick={e=>{this.importTestData()}}
        />
      </SpeedDial>


        <Dialog
          fullWidth={true}
          maxWidth='sm'
          open={this.state.openDialog}
          onClose={this.handleCloseDialog}
          scroll={'paper'}
          aria-labelledby='scroll-dialog-title'
        >
          <DialogTitle id='scroll-dialog-title'>Filter by</DialogTitle>
          <DialogContent>
            <FormControl component='fieldset' className={classes.formControl}>
              <FormLabel component='legend'>Level</FormLabel>
                {allLevels.map(level => {
                  return <FormControlLabel
                    key={level}
                    label={level}
                    checked={this.state.filteredLevels[level]}
                    control={
                      <Checkbox
                        color='default'
                        value={level}
                      />
                    }
                    onChange={e => {
                      this.onFilterFormChanged(
                        (e.target.checked) ? 'log-level-add' : 'log-level-remove',
                        level,
                      )
                    }}
                  />
                })}
            </FormControl>
            <FormControl component='fieldset' className={classes.formControl}>
              <FormLabel component='legend'>Message</FormLabel>
              <TextField
                id="filter-message"
                ref={this.filterrMessageRef}
                multiline
                rows="4"
                defaultValue={this.filters.message}
                className={classes.textField}
                margin="normal"
                variant="outlined"
                helperText="/<regular expression>/"
                onChange={e => {
                  this.onFilterFormChanged('message', e.target.value)
                }}
              />
            </FormControl>
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleCloseDialog} color='primary'>
              Close
            </Button>
            <Button onClick={e => {this.filter()}} color='primary'>
              Apply Filters
            </Button>
          </DialogActions>
        </Dialog>
    </div>
  }
}

const MyApp = withSnackbar(CenteredGrid);

class IntegrationNotistack extends React.Component {
  render() {
    return (
      <SnackbarProvider maxSnack={3} autoHideDuration={5000}>
        <MyApp {...this.props} />
      </SnackbarProvider>
    )
  }
}

class RecordDetail extends React.Component {
  state = {
    closed: true
  }

  toggle(open) {
    this.setState({'closed': !open})
  }

  render() {
    const { record, nodes, classes } = this.props;

    return <TableRow className={'row-detail'} key={record.id + 'detail'}>
      <TableCell colSpan={nodes.length + 1}>
      {this.state.closed ? (
        <React.Fragment />
      ) : (
        <Grid container className={classes.root} spacing={10}>
          <Grid item xs={5}>
            <Highlight className='json'>{JSON.stringify(record.basic(), null, 2)}</Highlight>
          </Grid>
          <Grid item xs={5}>
            <Highlight className='json'>{JSON.stringify(record.extra, null, 2)}</Highlight>
          </Grid>
        </Grid>
      )}
      </TableCell>
    </TableRow>
  }
}

export default withStyles(styles)(IntegrationNotistack)
