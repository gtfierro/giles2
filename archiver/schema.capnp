@0xef3606c9afc0887c;
using Go = import "go.capnp";
$Go.package("archiver");
$Go.import("testpkg");


struct SmapMessage2Capn { 
   path        @0:   Text; 
   uUID        @1:   Text; 
   properties  @2:   SmapProperties2Capn; 
   actuator    @3:   List(DictEntryCapn); 
   metadata    @4:   List(DictEntryCapn); 
   readings    @5:   List(SmapNumberReadingCapn); 
} 

struct SmapNumberReadingCapn { 
   time   @0:   UInt64; 
   value  @1:   Float64; 
} 

struct SmapProperties2Capn { 
   unitOfTime     @0:   UInt64; 
   unitOfMeasure  @1:   Text; 
   streamType     @2:   UInt64; 
} 

struct DictEntryCapn {
    key     @0: Text;
    value   @1: Text;
}

##compile with:

##
##
##   capnp compile -ogo ./schema.capnp

