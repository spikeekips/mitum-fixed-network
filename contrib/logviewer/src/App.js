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
import { unstable_Box as Box } from '@material-ui/core/Box';
import Chip from '@material-ui/core/Chip';
import Highlight from 'react-highlight'
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import Button from '@material-ui/core/Button';
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
    bottom: theme.spacing.unit * 2,
    right: theme.spacing.unit * 3,
  },
});


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
    levels: [],
    modules: [],
    msgs: [],
    record: null,
    speedDial: false,
    openDialog: false,
    moduleColors: [],
  }

  log = null
  prevRecords = []
  prevRecordsFragment = null

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
        nodes: this.log.nodes,
        msgs: this.log.msgs,
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

  handleCloseDialog = () => {
    this.setState({ openDialog: false, });
  };

  importTestData = () => {
      this.log = Log.load(raw)

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
        msgs: this.log.msgs,
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
      delimiter: ','
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

    this.log.records.map(record => {
      var row = [record.t.orig]

      this.log.nodes.map(node => {
        if (node === record.node) {
          row.push(JSON.stringify(record, null, "  "))
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

  toggleDetail(ref, o) {
    const tr = ReactDOM.findDOMNode(ref.current)

    var open = !tr.nextSibling.classList.contains('row-detail-open')

    if (open === true) {
      tr.nextSibling.classList.add('row-detail-open')
    } else {
      tr.nextSibling.classList.remove('row-detail-open')
    }

    o.current.toggle(open)

    return
  }

  sanitizeRecordMessage(message) {
    var a = markdown.markdown.toHTML(message)
    return a.slice(3, a.length - 4)
  }

  renderRecord(first, record, nodes) {
    const { classes } = this.props;

    var i = nodes.indexOf(record.node)
    if (i < 0) {
      return null
    }

    var rowRef = React.createRef()
    var rowDetailRef = React.createRef()

    return <React.Fragment key={record.id + 'f'}>
      <TableRow key={record.id} ref={rowRef}>
        <TableCell key={record.id + '-t'}>
          <IconButton className={classes.button} aria-label='Bookmark' onClick={e => {
            const tr = ReactDOM.findDOMNode(rowRef.current)
            tr.classList.toggle('selected')
            tr.nextSibling.classList.toggle('selected')

            this.toggleDetail(rowRef, rowDetailRef)
          }}>
            <BookmarksIcon />
          </IconButton>
          <Chip label={record.level} className={'lvl lvl-' + record.level} color='secondary' />
          <span className='t'>
            {record.t.elapsed(first.t)}
          </span>
        </TableCell>
        {nodes.map((node, index) => (
          <TableCell
            className={classes.listTableTd}
            key={record.id + node + '-m'}
            onClick={e => this.toggleDetail(rowRef, rowDetailRef)}
          >
          {i === index ? (
            <div key={record.id + record.module} className='record'>
              <Chip label={record.module} className={'module'} color='secondary' style={{
                backgroundColor: this.state.moduleColors[this.state.modules.indexOf(record.module)],
                color: fontColorByBG(this.state.moduleColors[this.state.modules.indexOf(record.module)]),
              }} />
              <span dangerouslySetInnerHTML={{__html: this.sanitizeRecordMessage(record.message)}} />
            </div>
            ) : (
              <Typography key={record.id + node + 'ty'}></Typography>
            )
          }
          </TableCell>
        ))}
      </TableRow>
      <RecordDetail classes={classes} nodes={nodes} record={record} ref={rowDetailRef} />
    </React.Fragment>
  }

  renderRecordsMore() {
    var records = this.log.records.slice(0, this.recordsOffset+this.limit)
    if (records.length < 1) {
      return
    }

    this.setState({ records: records })

    this.recordsOffset += this.limit
    this.onLoading = false
  }

  renderRecords(records, nodes) {
    const { classes } = this.props;

    var update = false
    if (records.length !== this.prevRecords.length) {
      update = true
    } else if (records.length > 0 && this.prevRecords.length > 0) {
      var pl = this.prevRecords[this.prevRecords.length - 1].t.n
      var tl = records[records.length - 1].t.n
      update = pl !== tl
    }

    if (!update && this.prevRecordsFragment != null) {
      return this.prevRecordsFragment
    }
 
    this.prevRecords = records
    this.prevRecordsFragment = <React.Fragment>
      <Box height='100%'>
        <Table id='inner-root' className={' scrollable'}>
          <TableBody>
            {this.state.records.map((record, index) => {
              if (index % 30 === 0) {
                return <React.Fragment key={record.id + 'r' + index}>
                  <TableRow className='header' key={record.id + 'h' + index}>
                    <TableCell className={classes.listTableT} key={'t'}><div>T</div></TableCell>
                    {this.state.nodes.map(node => (
                      <TableCell align='left' key={node}><div>{node}</div></TableCell>
                    ))}
                  </TableRow>
                  {this.renderRecord(records[0], record, nodes)}
                </React.Fragment>
              }

              return this.renderRecord(records[0], record, nodes)
            })}
          </TableBody>
        </Table>
      </Box>
    </React.Fragment>

    return this.prevRecordsFragment
  }

  shouldComponentUpdate(nextProps, nextState) {
    if (nextState.speedDial !== this.state.speedDial) {
      return true
    }

    if (nextState.records.length < 1 || this.state.records.length < 1) {
      this.prevRecords = this.state.records
      return true
    }

    if (nextState.records.length !== this.state.records.length) {
      this.prevRecords = this.state.records
      return true
    }

    var pl = this.state.records[this.state.records.length - 1].t.n
    var tl = nextState.records[nextState.records.length - 1].t.n
    if (pl === tl) {
      return false
    }

    //this.prevRecords = this.state.records

    return true
  }

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

      {this.renderRecords(this.state.records, this.state.nodes)}

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
      {this.state.msgs.length > 0 ? (
        <SpeedDialAction
          key={'expand-collapse-all'}
          icon={<SettingsOverscanIcon />}
          tooltipTitle={'expand/collapse all'}
          tooltipOpen
          onClick={e=>{this.toggleExpandAll()}}
        />) : ([]) }
      {this.state.msgs.length > 0 ? (
        <SpeedDialAction
          key={'filter'}
          icon={<FilterListIcon />}
          tooltipTitle={'filter records'}
          tooltipOpen
          onClick={e=>{this.setState({openDialog: true})}}
        />) : ([]) }
      {this.state.msgs.length > 0 ? (
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
                {this.state.levels.map(level => (
                  <FormControlLabel
                    key={level}
                    label={level}
                    control={ <Checkbox color='default' value={level} /> }
                  />
                ))}
            </FormControl>
            <FormControl component='fieldset' className={classes.formControl}>
              <FormLabel component='legend'>Messages</FormLabel>
                {this.state.msgs.map(msg => (
                  <FormControlLabel
                    key={msg}
                    label={msg}
                    control={ <Checkbox color='default' value={msg} /> }
                  />
                ))}
            </FormControl>
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleCloseDialog} color='primary'>
              Close
            </Button>
            <Button onClick={this.handleCloseDialog} color='primary'>
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
        <Grid container className={classes.root} spacing={16}>
          <Grid item xs={4}>
            <Highlight className='json'>{JSON.stringify(record.basic(), null, 2)}</Highlight>
          </Grid>
          <Grid item xs={8}>
            <Highlight className='json'>{JSON.stringify(record.extra, null, 2)}</Highlight>
          </Grid>
        </Grid>
      )}
      </TableCell>
    </TableRow>
  }
}

export default withStyles(styles)(IntegrationNotistack)
