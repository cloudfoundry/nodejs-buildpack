//===========================================================================

#ifndef BSON_H_
#define BSON_H_

//===========================================================================

#ifdef __arm__
#define USE_MISALIGNED_MEMORY_ACCESS 0
#else
#define USE_MISALIGNED_MEMORY_ACCESS 1
#endif

#include <node.h>
#include <node_object_wrap.h>
#include <v8.h>
#include "nan.h"

using v8::Local;
using v8::Value;
using v8::Integer;
using v8::Number;
using v8::Int32;
using v8::Uint32;
using v8::String;
using v8::Object;
using v8::Date;
using v8::Array;
using v8::RegExp;
using v8::Function;
using v8::FunctionTemplate;
using Nan::Persistent;
using Nan::ObjectWrap;

#define NanStr(x) (Unmaybe(Nan::New<String>(x)))
#define NanHas(obj, key) (Nan::Has(obj, NanKey(key)).FromJust())
#define NanGet(obj, key) (Unmaybe(Nan::Get(obj, NanKey(key))))
// Unmaybe overloading to conviniently convert from Local/MaybeLocal/Maybe to Local/plain value
template <class T>
inline Local<T> Unmaybe(Local<T> h) {
    return h;
}
template <class T>
inline Local<T> Unmaybe(Nan::MaybeLocal<T> h) {
    assert(!h.IsEmpty());
    return h.ToLocalChecked();
}
template <class T>
inline T Unmaybe(Nan::Maybe<T> h) {
    assert(h.IsJust());
    return h.FromJust();
}
// NanKey overloading to conviniently convert to a propert key for object/array
inline int NanKey(int i) {
    return i;
}
inline Local<String> NanKey(const char* s) {
    return NanStr(s);
}
inline Local<String> NanKey(const std::string& s) {
    return NanStr(s);
}
inline Local<String> NanKey(const Local<String>& s) {
    return s;
}
inline Local<String> NanKey(const Nan::Persistent<String>& s) {
    return NanStr(s);
}

//===========================================================================

enum BsonType
{
	BSON_TYPE_NUMBER		= 1,
	BSON_TYPE_STRING		= 2,
	BSON_TYPE_OBJECT		= 3,
	BSON_TYPE_ARRAY			= 4,
	BSON_TYPE_BINARY		= 5,
	BSON_TYPE_UNDEFINED		= 6,
	BSON_TYPE_OID			= 7,
	BSON_TYPE_BOOLEAN		= 8,
	BSON_TYPE_DATE			= 9,
	BSON_TYPE_NULL			= 10,
	BSON_TYPE_REGEXP		= 11,
	BSON_TYPE_CODE			= 13,
	BSON_TYPE_SYMBOL		= 14,
	BSON_TYPE_CODE_W_SCOPE	= 15,
	BSON_TYPE_INT			= 16,
	BSON_TYPE_TIMESTAMP		= 17,
	BSON_TYPE_LONG			= 18,
	BSON_TYPE_MAX_KEY		= 0x7f,
	BSON_TYPE_MIN_KEY		= 0xff
};

//===========================================================================

template<typename T> class BSONSerializer;

class BSON : public Nan::ObjectWrap {
public:
	BSON();
	~BSON();

	static void Initialize(Local<Object> target);
 	static NAN_METHOD(BSONDeserializeStream);

	// JS based objects
	static NAN_METHOD(BSONSerialize);
	static NAN_METHOD(BSONDeserialize);

        // Calculate size of function
	static NAN_METHOD(CalculateObjectSize);
	static NAN_METHOD(SerializeWithBufferAndIndex);

	// Constructor used for creating new BSON objects from C++
	static Persistent<FunctionTemplate> constructor_template;

public:
	Persistent<Object> buffer;
	size_t maxBSONSize;

private:
	static NAN_METHOD(New);
	static Local<Value> deserialize(BSON *bson, char *data, uint32_t dataLength, uint32_t startIndex, bool is_array_item);

	// BSON type instantiate functions
	Persistent<Function> longConstructor;
	Persistent<Function> objectIDConstructor;
	Persistent<Function> binaryConstructor;
	Persistent<Function> codeConstructor;
	Persistent<Function> dbrefConstructor;
	Persistent<Function> symbolConstructor;
	Persistent<Function> doubleConstructor;
	Persistent<Function> timestampConstructor;
	Persistent<Function> minKeyConstructor;
	Persistent<Function> maxKeyConstructor;

	Local<Object> GetSerializeObject(const Local<Value>& object);

	template<typename T> friend class BSONSerializer;
	friend class BSONDeserializer;
};

//===========================================================================

class CountStream
{
public:
	CountStream() : count(0) { }

	void	WriteByte(int value)									{ ++count; }
	void	WriteByte(const Local<Object>&, const Local<String>&)	{ ++count; }
	void	WriteBool(const Local<Value>& value)					{ ++count; }
	void	WriteInt32(int32_t value)								{ count += 4; }
	void	WriteInt32(const Local<Value>& value)					{ count += 4; }
	void	WriteInt32(const Local<Object>& object, const Local<String>& key) { count += 4; }
	void	WriteInt64(int64_t value)								{ count += 8; }
	void	WriteInt64(const Local<Value>& value)					{ count += 8; }
	void	WriteDouble(double value)								{ count += 8; }
	void	WriteDouble(const Local<Value>& value)					{ count += 8; }
	void	WriteDouble(const Local<Object>&, const Local<String>&) { count += 8; }
	void	WriteUInt32String(uint32_t name)						{ char buffer[32]; count += sprintf(buffer, "%u", name) + 1; }
	void	WriteLengthPrefixedString(const Local<String>& value)	{ count += value->Utf8Length()+5; }
	void	WriteObjectId(const Local<Object>& object, const Local<String>& key)				{ count += 12; }
	void	WriteString(const Local<String>& value)					{ count += value->Utf8Length() + 1; }	// This returns the number of bytes exclusive of the NULL terminator
	void	WriteData(const char* data, size_t length)				{ count += length; }

	void*	BeginWriteType()										{ ++count; return NULL; }
	void	CommitType(void*, BsonType)								{ }
	void*	BeginWriteSize()										{ count += 4; return NULL; }
	void	CommitSize(void*)										{ }

	size_t GetSerializeSize() const									{ return count; }

	// Do nothing. CheckKey is implemented for DataStream
	void	CheckKey(const Local<String>&)							{ }

private:
	size_t	count;
};

const size_t MAX_BSON_SIZE (1024*1024*17);

class DataStream
{
public:
	DataStream(char* aDestinationBuffer) : destinationBuffer(aDestinationBuffer), p(aDestinationBuffer) { }

	void	WriteByte(int value) {
		if((size_t)((p + 1) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		*p++ = value;
	}

	void	WriteByte(const Local<Object>& object, const Local<String>& key)	{
		if((size_t)((p + 1) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		*p++ = object->Get(key)->Int32Value();
	}

#if USE_MISALIGNED_MEMORY_ACCESS
	void	WriteInt32(int32_t value)	{
		if((size_t)((p + 4) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		*reinterpret_cast<int32_t*>(p) = value;
		p += 4;
	}

	void	WriteInt64(int64_t value)	{
		if((size_t)((p + 8) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		*reinterpret_cast<int64_t*>(p) = value;
		p += 8;
	}

	void	WriteDouble(double value) {
		if((size_t)((p + 8) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		*reinterpret_cast<double*>(p) = value;
		p += 8;
	}
#else
	void	WriteInt32(int32_t value)	{
		if((size_t)((p + 4) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		memcpy(p, &value, 4);
		p += 4;
	}

	void	WriteInt64(int64_t value)	{
		if((size_t)((p + 8) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		memcpy(p, &value, 8);
		p += 8;
	}

	void	WriteDouble(double value) {
		if((size_t)((p + 8) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		memcpy(p, &value, 8);
		p += 8;
	}
#endif
	void	WriteBool(const Local<Value>& value) {
		WriteByte(value->BooleanValue() ? 1 : 0);
	}

	void	WriteInt32(const Local<Value>& value) {
		WriteInt32(value->Int32Value());
	}

	void	WriteInt32(const Local<Object>& object, const Local<String>& key) {
		WriteInt32(object->Get(key));
	}

	void	WriteInt64(const Local<Value>& value) {
		WriteInt64(value->IntegerValue());
	}

	void	WriteDouble(const Local<Value>& value)	{
		WriteDouble(value->NumberValue());
	}

	void	WriteDouble(const Local<Object>& object, const Local<String>& key) {
		WriteDouble(object->Get(key));
	}

	void	WriteUInt32String(uint32_t name) {
		p += sprintf(p, "%u", name) + 1;
	}

	void	WriteLengthPrefixedString(const Local<String>& value)	{
		int32_t length = value->Utf8Length()+1;
		if((size_t)((p + length) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		WriteInt32(length);
		WriteString(value);
	}

	void	WriteObjectId(const Local<Object>& object, const Local<String>& key);

	void	WriteString(const Local<String>& value)	{
		if((size_t)(p - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		p += value->WriteUtf8(p);
	}		// This returns the number of bytes inclusive of the NULL terminator.

	void	WriteData(const char* data, size_t length) {
		if((size_t)((p + length) - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		memcpy(p, data, length);
		p += length;
	}

	void*	BeginWriteType()										{
		void* returnValue = p; p++;
		return returnValue;
	}

	void	CommitType(void* beginPoint, BsonType value) {
		*reinterpret_cast<unsigned char*>(beginPoint) = value;
	}

	void*	BeginWriteSize() {
		if((size_t)(p - destinationBuffer) > MAX_BSON_SIZE) throw "document is larger than max bson document size of 16MB";
		void* returnValue = p; p += 4;
		return returnValue;
	}

#if USE_MISALIGNED_MEMORY_ACCESS
	void	CommitSize(void* beginPoint) {
		*reinterpret_cast<int32_t*>(beginPoint) = (int32_t) (p - (char*) beginPoint);
	}
#else
	void	CommitSize(void* beginPoint) {
		int32_t value = (int32_t) (p - (char*) beginPoint);
		memcpy(beginPoint, &value, 4);
	}
#endif

	size_t GetSerializeSize() const	{
		return p - destinationBuffer;
	}

	void	CheckKey(const Local<String>& keyName);

public:
	char *const	destinationBuffer;		// base, never changes
	char*	p;													// cursor into buffer
};

template<typename T> class BSONSerializer : public T
{
private:
	typedef T Inherited;

public:
	BSONSerializer(BSON* aBson, bool aCheckKeys, bool aSerializeFunctions) : Inherited(), checkKeys(aCheckKeys), serializeFunctions(aSerializeFunctions), bson(aBson) { }
	BSONSerializer(BSON* aBson, bool aCheckKeys, bool aSerializeFunctions, char* parentParam) : Inherited(parentParam), checkKeys(aCheckKeys), serializeFunctions(aSerializeFunctions), bson(aBson) { }

	void SerializeDocument(const Local<Value>& value);
	void SerializeArray(const Local<Value>& value);
	void SerializeValue(void* typeLocation, const Local<Value> value, bool isArray);

private:
	bool		checkKeys;
	bool		serializeFunctions;
	BSON*		bson;
};

//===========================================================================

class BSONDeserializer
{
public:
	BSONDeserializer(BSON* aBson, char* data, size_t length);
	BSONDeserializer(BSONDeserializer& parentSerializer, size_t length);

	Local<Value> DeserializeDocument(bool promoteLongs);

	bool			HasMoreData() const { return p < pEnd; }
	Local<Value>	ReadCString();
	uint32_t		ReadIntegerString();
	int32_t			ReadRegexOptions();
	Local<String>	ReadString();
	Local<String>	ReadObjectId();

	unsigned char	ReadByte()			{ return *reinterpret_cast<unsigned char*>(p++); }
#if USE_MISALIGNED_MEMORY_ACCESS
	int32_t			ReadInt32()			{ int32_t returnValue = *reinterpret_cast<int32_t*>(p); p += 4; return returnValue; }
	uint32_t		ReadUInt32()		{ uint32_t returnValue = *reinterpret_cast<uint32_t*>(p); p += 4; return returnValue; }
	int64_t			ReadInt64()			{ int64_t returnValue = *reinterpret_cast<int64_t*>(p); p += 8; return returnValue; }
	double			ReadDouble()		{ double returnValue = *reinterpret_cast<double*>(p); p += 8; return returnValue; }
#else
	int32_t			ReadInt32()			{ int32_t returnValue; memcpy(&returnValue, p, 4); p += 4; return returnValue; }
	uint32_t		ReadUInt32()		{ uint32_t returnValue; memcpy(&returnValue, p, 4); p += 4; return returnValue; }
	int64_t			ReadInt64()			{ int64_t returnValue; memcpy(&returnValue, p, 8); p += 8; return returnValue; }
	double			ReadDouble()		{ double returnValue; memcpy(&returnValue, p, 8); p += 8; return returnValue; }
#endif

	size_t			GetSerializeSize() const { return p - pStart; }

private:
	Local<Value> DeserializeArray(bool promoteLongs);
	Local<Value> DeserializeValue(BsonType type, bool promoteLongs);
	Local<Value> DeserializeDocumentInternal(bool promoteLongs);
	Local<Value> DeserializeArrayInternal(bool promoteLongs);

	BSON*		bson;
	char* const pStart;
	char*		p;
	char* const	pEnd;
};

//===========================================================================

#endif  // BSON_H_

//===========================================================================
