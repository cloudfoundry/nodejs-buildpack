module SemanticRange
  class Comparator
    attr_reader :semver, :operator, :value
    def initialize(comp, loose)
      if comp.is_a?(Comparator)
        return comp if comp.loose == loose
        @comp = comp.value
      end

      @loose = loose
      parse(comp)

      @value = @semver == ANY ? '' : @operator + @semver.version
    end

    def to_s
      @value
    end

    def test(version)
      return true if @semver == ANY
      version = Version.new(version, @loose) if version.is_a?(String)
      SemanticRange.cmp(version, @operator, @semver, @loose)
    end

    def parse(comp)
      m = comp.match(@loose ? COMPARATORLOOSE : COMPARATOR)
      raise InvalidComparator.new(comp) unless m

      @operator = m[1]
      @operator = '' if @operator == '='

      @semver = !m[2] ? ANY : Version.new(m[2], @loose)
    end
  end
end
