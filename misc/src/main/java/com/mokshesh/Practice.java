package com.mokshesh;

import java.time.LocalDateTime;
import java.time.temporal.ChronoUnit;

public class Practice {
    private String name;
    private int id;

    Practice(){

    }

    Practice(int id, String name){        
        this.name = name;
        this.id = id;
    }

    public static PracticeBuilder builder(){
        return new PracticeBuilder();
    }

    static class PracticeBuilder {
        private String name;
        private int id;
        public PracticeBuilder id(int id){
            this.id = id;
            return this;
        }
        public PracticeBuilder name(String name){
            this.name = name;
            return this;
        }
        public Practice build(){
            return new Practice(this.id, name);
        }

    }

    enum VehicleType {
        CAR, BIKE
    }
    public static void main(String[] args) {
        ChronoUnit.DAYS.between(LocalDateTime.now(), LocalDateTime.now());
        Practice practice = Practice.builder().id(1).name("abc").build();
    }
    

}
