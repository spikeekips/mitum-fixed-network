class LogFilter {
  constructor(records, filters) {
    this.all = records
    this.offset = 0
    this.defaultFilters = {
			levels: ['eror', 'dbug', 'info', 'warn', 'crit'],
      message: null,
    }
		this.filters = this.defaultFilters
    this.setFilters(filters)
  }

  setFilters(filters) {
    if (filters === null || Object.keys(filters).length < 1) {
      filters = this.defaultFilters
    }

    this.filters = filters

    if (this.filters.message === null || this.filters.message.trim().length < 1) {
      this.filters.message = ""
      this.filters.regexp = null
    } else {
      if (this.filters.message.search(/^\//) < 0) {
        this.filters.regexp = new RegExp(this.filters.message)
      } else {
        try {
          this.filters.regexp = new RegExp(eval(this.filters.message))
        } catch (e) {
          console.error(e)
          return null
        }
      }

      this.filters.message = this.filters.message.trim()
    }
    this.offset = 0
  }

  reset() {
		this.offset = 0
	}

  *filter(limit) {
    if (this.all.length <= this.offset) {
      return []
    }

		var yielt = 0
    var count = 0
    for (let record of this.all.slice(this.offset)) {
			if (yielt > limit) {
				break
			}
      if (!this.filterLevels(record)) {
        count++
        continue
      }
      if (!this.filterMessage(record)) {
        count++
        continue
      }
      yield record
      yielt++
      count++
    }

    this.offset += count
  }

  filterLevels(record) {
    if (this.filters.levels.length < 1) {
      return true
    }
    if (!this.filters.levels.includes(record.level)) {
      return false
    }

    return true
  }

  filterMessage(record) {
    if (this.filters.regexp === null) {
      return true
    }

    if (!this.filters.regexp.test(record.body)) {
      return false
    }

    return true
  }
}

export default LogFilter
